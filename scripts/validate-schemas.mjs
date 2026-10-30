#!/usr/bin/env node
/**
 * scripts/validate-schemas.mjs
 * Validates Soundcheck JSON schemas and testdata fixtures.
 * Zero-dependency standalone runner with optional ajv-cli cross-check.
 */

import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const REPO_ROOT = path.resolve(__dirname, '..');

// ---------------------------------------------------------------------------
// 1. Lightweight YAML Parser for agent-doctor.yaml
// ---------------------------------------------------------------------------
export function parseConfigYaml(content) {
  const root = {};
  const lines = content.split('\n');
  const stack = [{ indent: -1, target: root }];

  for (let i = 0; i < lines.length; i++) {
    const raw = lines[i];
    const trimmed = raw.trim();
    if (!trimmed || trimmed.startsWith('#')) continue;

    const indent = raw.search(/\S/);

    while (stack.length > 1 && stack[stack.length - 1].indent >= indent) {
      stack.pop();
    }
    const current = stack[stack.length - 1].target;

    if (trimmed.startsWith('- ')) {
      const valStr = trimmed.slice(2).trim();
      const val = parseScalar(valStr);
      if (Array.isArray(current)) {
        current.push(val);
      }
      continue;
    }

    const colonIdx = trimmed.indexOf(':');
    if (colonIdx === -1) continue;

    const key = trimmed.slice(0, colonIdx).trim();
    const rest = trimmed.slice(colonIdx + 1).trim();

    if (!rest) {
      let isList = false;
      for (let j = i + 1; j < lines.length; j++) {
        const nextTrim = lines[j].trim();
        if (!nextTrim || nextTrim.startsWith('#')) continue;
        isList = nextTrim.startsWith('- ');
        break;
      }
      const newTarget = isList ? [] : {};
      current[key] = newTarget;
      stack.push({ indent, target: newTarget });
    } else {
      current[key] = parseScalar(rest);
    }
  }

  return root;
}

function parseScalar(str) {
  if (str.startsWith('[') && str.endsWith(']')) {
    const inner = str.slice(1, -1).trim();
    if (!inner) return [];
    return inner.split(',').map((s) => parseScalar(s.trim()));
  }
  if (str === 'true') return true;
  if (str === 'false') return false;
  if (str === 'null') return null;
  if (!isNaN(Number(str)) && str !== '') return Number(str);
  return str.replace(/^["']|["']$/g, '');
}

// ---------------------------------------------------------------------------
// 2. Draft-07 JSON Schema Validator
// ---------------------------------------------------------------------------
export class SchemaValidator {
  constructor(schemas = {}) {
    this.schemas = schemas;
  }

  addSchema(id, schema) {
    this.schemas[id] = schema;
    if (schema.$id) this.schemas[schema.$id] = schema;
  }

  validate(schema, data, rootSchema = schema) {
    const errors = [];
    this._validateNode(schema, data, '#', errors, rootSchema);
    return {
      valid: errors.length === 0,
      errors
    };
  }

  _resolveRef(ref, rootSchema) {
    if (ref.startsWith('#/')) {
      const parts = ref.slice(2).split('/');
      let curr = rootSchema;
      for (const p of parts) {
        if (!curr || typeof curr !== 'object') return null;
        curr = curr[p];
      }
      return curr;
    }
    if (this.schemas[ref]) return this.schemas[ref];
    return null;
  }

  _validateNode(schema, data, pathStr, errors, rootSchema) {
    if (!schema || typeof schema !== 'object') return;

    if (schema.$ref) {
      const target = this._resolveRef(schema.$ref, rootSchema);
      if (!target) {
        errors.push(`${pathStr}: unresolved $ref ${schema.$ref}`);
        return;
      }
      this._validateNode(target, data, pathStr, errors, rootSchema);
      return;
    }

    if (schema.oneOf) {
      let matchCount = 0;
      const subErrors = [];
      for (let i = 0; i < schema.oneOf.length; i++) {
        const branchErrors = [];
        this._validateNode(schema.oneOf[i], data, `${pathStr}/oneOf[${i}]`, branchErrors, rootSchema);
        if (branchErrors.length === 0) matchCount++;
        else subErrors.push(branchErrors.join('; '));
      }
      if (matchCount !== 1) {
        errors.push(`${pathStr}: expected exactly 1 oneOf match, got ${matchCount}. (${subErrors.slice(0, 2).join(' | ')})`);
      }
      return;
    }

    if (schema.anyOf) {
      let matchCount = 0;
      for (let i = 0; i < schema.anyOf.length; i++) {
        const branchErrors = [];
        this._validateNode(schema.anyOf[i], data, `${pathStr}/anyOf[${i}]`, branchErrors, rootSchema);
        if (branchErrors.length === 0) {
          matchCount++;
          break;
        }
      }
      if (matchCount === 0) {
        errors.push(`${pathStr}: failed to match any anyOf branch`);
      }
    }

    if (schema.allOf) {
      for (let i = 0; i < schema.allOf.length; i++) {
        this._validateNode(schema.allOf[i], data, `${pathStr}/allOf[${i}]`, errors, rootSchema);
      }
    }

    if (schema.enum && !schema.enum.includes(data)) {
      errors.push(`${pathStr}: value ${JSON.stringify(data)} not in enum [${schema.enum.join(', ')}]`);
    }

    if (schema.type) {
      const types = Array.isArray(schema.type) ? schema.type : [schema.type];
      const actualType = getActualType(data);
      const matches = types.some((t) => t === actualType || (t === 'number' && actualType === 'integer'));
      if (!matches) {
        errors.push(`${pathStr}: expected type [${types.join('|')}], got ${actualType}`);
        return;
      }
    }

    if (typeof data === 'number') {
      if (schema.minimum !== undefined && data < schema.minimum) {
        errors.push(`${pathStr}: ${data} < minimum ${schema.minimum}`);
      }
      if (schema.maximum !== undefined && data > schema.maximum) {
        errors.push(`${pathStr}: ${data} > maximum ${schema.maximum}`);
      }
    }

    if (Array.isArray(data)) {
      if (schema.minItems !== undefined && data.length < schema.minItems) {
        errors.push(`${pathStr}: length ${data.length} < minItems ${schema.minItems}`);
      }
      if (schema.items) {
        for (let idx = 0; idx < data.length; idx++) {
          this._validateNode(schema.items, data[idx], `${pathStr}[${idx}]`, errors, rootSchema);
        }
      }
    }

    if (data && typeof data === 'object' && !Array.isArray(data)) {
      if (schema.required) {
        for (const req of schema.required) {
          if (!(req in data)) {
            errors.push(`${pathStr}: missing required field "${req}"`);
          }
        }
      }

      const definedKeys = new Set(Object.keys(schema.properties || {}));
      for (const [key, val] of Object.entries(data)) {
        if (schema.properties && schema.properties[key]) {
          this._validateNode(schema.properties[key], val, `${pathStr}.${key}`, errors, rootSchema);
        } else if (schema.additionalProperties === false && !definedKeys.has(key)) {
          errors.push(`${pathStr}: unknown additional property "${key}"`);
        } else if (schema.additionalProperties && typeof schema.additionalProperties === 'object') {
          this._validateNode(schema.additionalProperties, val, `${pathStr}.${key}`, errors, rootSchema);
        }
      }
    }
  }
}

function getActualType(val) {
  if (val === null) return 'null';
  if (Array.isArray(val)) return 'array';
  if (typeof val === 'number') return Number.isInteger(val) ? 'integer' : 'number';
  return typeof val;
}

// ---------------------------------------------------------------------------
// 3. Verification Suite
// ---------------------------------------------------------------------------
async function main() {
  console.log('--- Soundcheck Schema Validation Suite ---');
  let passCount = 0;
  let failCount = 0;

  function assert(name, condition, details = '') {
    if (condition) {
      console.log(`  ✓ ${name}`);
      passCount++;
    } else {
      console.error(`  ✗ ${name}`);
      if (details) console.error(`    Details: ${details}`);
      failCount++;
    }
  }

  // Load Schemas
  const schemasDir = path.join(REPO_ROOT, 'schemas');
  const schemaFiles = [
    'config.schema.json',
    'probe_result.schema.json',
    'classification.schema.json',
    'assignment.schema.json',
    'manifest.schema.json',
    'event.schema.json'
  ];

  const validator = new SchemaValidator();
  const loadedSchemas = {};

  for (const sf of schemaFiles) {
    const fullPath = path.join(schemasDir, sf);
    const exists = fs.existsSync(fullPath);
    assert(`Schema file exists: schemas/${sf}`, exists);
    if (exists) {
      const parsed = JSON.parse(fs.readFileSync(fullPath, 'utf8'));
      assert(`Schema valid JSON: schemas/${sf}`, !!parsed.$schema && !!parsed.$id);
      validator.addSchema(sf, parsed);
      loadedSchemas[sf] = parsed;
    }
  }

  // 1. Validate agent-doctor.yaml
  const yamlPath = path.join(REPO_ROOT, 'agent-doctor.yaml');
  const yamlContent = fs.readFileSync(yamlPath, 'utf8');
  const configData = parseConfigYaml(yamlContent);
  const configRes = validator.validate(loadedSchemas['config.schema.json'], configData);
  assert(
    'agent-doctor.yaml matches config.schema.json',
    configRes.valid,
    configRes.errors.join(', ')
  );

  // 2. Validate testdata/golden/resolve.json against assignment.schema.json
  const resolvePath = path.join(REPO_ROOT, 'testdata/golden/resolve.json');
  const resolveData = JSON.parse(fs.readFileSync(resolvePath, 'utf8'));
  const resolveRes = validator.validate(loadedSchemas['assignment.schema.json'], resolveData);
  assert(
    'testdata/golden/resolve.json matches assignment.schema.json',
    resolveRes.valid,
    resolveRes.errors.join(', ')
  );

  // Also validate each role assignment individually
  for (const [role, assign] of Object.entries(resolveData)) {
    const rRes = validator.validate(loadedSchemas['assignment.schema.json'], assign);
    assert(
      `resolve.json role "${role}" matches assignment.schema.json`,
      rRes.valid,
      rRes.errors.join(', ')
    );
  }

  // 3. Validate testdata/golden/manifest.json against manifest.schema.json
  const manifestPath = path.join(REPO_ROOT, 'testdata/golden/manifest.json');
  const manifestData = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
  const manifestRes = validator.validate(loadedSchemas['manifest.schema.json'], manifestData);
  assert(
    'testdata/golden/manifest.json matches manifest.schema.json',
    manifestRes.valid,
    manifestRes.errors.join(', ')
  );

  // 4. Validate testdata/golden/classify.json against classification.schema.json
  const classifyPath = path.join(REPO_ROOT, 'testdata/golden/classify.json');
  const classifyData = JSON.parse(fs.readFileSync(classifyPath, 'utf8'));
  const classifyRes = validator.validate(loadedSchemas['classification.schema.json'], classifyData);
  assert(
    'testdata/golden/classify.json matches classification.schema.json',
    classifyRes.valid,
    classifyRes.errors.join(', ')
  );

  // Also validate each classification output individually
  for (const item of classifyData) {
    const cRes = validator.validate(loadedSchemas['classification.schema.json'], item.classification);
    assert(
      `classify.json item "${item.id}" classification output matches schema`,
      cRes.valid,
      cRes.errors.join(', ')
    );
  }

  // 5. Validate testdata/golden/host_probe.json against probe_result.schema.json
  const hostProbePath = path.join(REPO_ROOT, 'testdata/golden/host_probe.json');
  const hostProbeData = JSON.parse(fs.readFileSync(hostProbePath, 'utf8'));
  const hostProbeRes = validator.validate(loadedSchemas['probe_result.schema.json'], hostProbeData);
  assert(
    'testdata/golden/host_probe.json matches probe_result.schema.json',
    hostProbeRes.valid,
    hostProbeRes.errors.join(', ')
  );

  // 6. Validate synthetic L0-L3 ProbeResults against probe_result.schema.json
  const sampleProbes = [
    {
      level: 'L0',
      status: 'passed',
      errorClass: 'ok',
      message: 'Binary found at /usr/local/bin/claude (v2.1.302)',
      latencyMs: 12,
      rawDetails: { exitCode: 0, stdout: '2.1.302' }
    },
    {
      level: 'L1',
      status: 'failed',
      errorClass: 'unauthenticated',
      message: 'Anthropic API key missing or expired',
      latencyMs: 45,
      rawDetails: { exitCode: 1, stderr: 'Unauthorized 401' }
    },
    {
      level: 'L2',
      status: 'passed',
      errorClass: 'ok',
      message: 'Smoke execution succeeded',
      latencyMs: 1200,
      rawDetails: { exitCode: 0, stdout: 'ok' }
    },
    {
      level: 'L3',
      status: 'passed',
      errorClass: 'ok',
      message: 'Models verified: [claude-sonnet-5-5]',
      latencyMs: 340
    }
  ];

  const probeArrayRes = validator.validate(loadedSchemas['probe_result.schema.json'], sampleProbes);
  assert(
    'sample L0-L3 probe results array matches probe_result.schema.json',
    probeArrayRes.valid,
    probeArrayRes.errors.join(', ')
  );

  for (const p of sampleProbes) {
    const singleRes = validator.validate(loadedSchemas['probe_result.schema.json'], p);
    assert(
      `sample ${p.level} probe result matches probe_result.schema.json`,
      singleRes.valid,
      singleRes.errors.join(', ')
    );
  }

  // 7. Validate synthetic streaming NDJSON events against event.schema.json
  const sampleEvents = [
    {
      type: 'probe_start',
      adapter: 'claude',
      level: 'L0',
      description: 'Checking claude binary in PATH'
    },
    {
      type: 'probe_done',
      adapter: 'claude',
      level: 'L0',
      status: 'passed',
      latencyMs: 14,
      message: 'claude 2.1.310 detected'
    },
    {
      type: 'log',
      level: 'info',
      message: 'Starting preflight doctor probe pool'
    },
    {
      type: 'terminal_chunk',
      stream: 'stdout',
      data: 'Hello from runner\n'
    }
  ];

  const eventArrayRes = validator.validate(loadedSchemas['event.schema.json'], sampleEvents);
  assert(
    'sample NDJSON events array matches event.schema.json',
    eventArrayRes.valid,
    eventArrayRes.errors.join(', ')
  );

  for (const ev of sampleEvents) {
    const singleEvRes = validator.validate(loadedSchemas['event.schema.json'], ev);
    assert(
      `sample event "${ev.type}" matches event.schema.json`,
      singleEvRes.valid,
      singleEvRes.errors.join(', ')
    );
  }

  // 8. Cross-validate with ajv-cli if available
  const ajvCheck = spawnSync('npx', ['--yes', 'ajv-cli', 'help'], { encoding: 'utf8' });
  if (ajvCheck.status === 0) {
    console.log(`\n  [ajv-cli available] Running ajv-cli cross-checks...`);
    const crossChecks = [
      { s: 'manifest.schema.json', d: 'testdata/golden/manifest.json' },
      { s: 'assignment.schema.json', d: 'testdata/golden/resolve.json' },
      { s: 'classification.schema.json', d: 'testdata/golden/classify.json' },
      { s: 'probe_result.schema.json', d: 'testdata/golden/host_probe.json' }
    ];

    for (const cc of crossChecks) {
      const res = spawnSync(
        'npx',
        [
          '--yes',
          'ajv-cli',
          'validate',
          '-s',
          path.join(schemasDir, cc.s),
          '-d',
          path.join(REPO_ROOT, cc.d)
        ],
        { encoding: 'utf8' }
      );
      assert(`ajv-cli validate: ${cc.d} with ${cc.s}`, res.status === 0, res.stderr || res.stdout);
    }

    // Cross-validate parsed agent-doctor.yaml with ajv-cli
    const tmpConfigJson = path.join(REPO_ROOT, 'testdata/golden/.tmp_config.json');
    fs.writeFileSync(tmpConfigJson, JSON.stringify(configData, null, 2), 'utf8');
    try {
      const res = spawnSync(
        'npx',
        [
          '--yes',
          'ajv-cli',
          'validate',
          '-s',
          path.join(schemasDir, 'config.schema.json'),
          '-d',
          tmpConfigJson
        ],
        { encoding: 'utf8' }
      );
      assert('ajv-cli validate: agent-doctor.yaml (parsed) with config.schema.json', res.status === 0, res.stderr || res.stdout);
    } finally {
      if (fs.existsSync(tmpConfigJson)) fs.unlinkSync(tmpConfigJson);
    }
  }

  console.log(`\nResults: ${passCount} passed, ${failCount} failed.`);
  if (failCount > 0) {
    process.exit(1);
  }
}

main().catch((err) => {
  console.error('Validation error:', err);
  process.exit(1);
});
