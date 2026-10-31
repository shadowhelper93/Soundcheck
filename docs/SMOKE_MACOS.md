[1/7] Building soundcheck...

(!) Some chunks are larger than 500 kB after minification. Consider:
- Using dynamic import() to code-split the application
- Use build.rollupOptions.output.manualChunks to improve chunking: https://rollupjs.org/configuration-options/#output-manualchunks
- Adjust chunk size limit for this warning via build.chunkSizeWarningLimit.
[2/7] Starting server on port 25403...
[3/7] /api/health ................
{"status":"ok","version":"1.4.0"}
ok
[4/7] /api/v1/doctor ..............
ok
[5/7] /api/v1/config .............
{"version":1,"roles":{"auxiliary":["claude:claude-haiku-5-5","codex:*","agy:gemini-2.5-flash"],"primary":["claude:claude-sonnet-5-5","claude:claude-opus-5-5","codex:*","opencode:anthropic/claude-sonnet-5-5","agy:gemini-2.5-pro"],"reviewer":["codex:*","agy:*","opencode:*","copilot:*","claude:*"],"subagent":["claude:claude-sonnet-5-5","codex:*","opencode:*","agy:gemini-2.5-flash"]},"policy":{"require_distinct_vendor":["primary","reviewer"],"max_cost_usd_per_run":5,"probe":{"deep":false,"ttl_minutes":30}}}
ok
[6/7] POST /api/v1/config/validate
200 ok
[7/7] bad PUT /api/v1/config (expect 400)
UI index:
<!doctype html>
<html lang="en" class="dark">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" href="data:imag

SMOKE PASS: all endpoints verified on port 25403
