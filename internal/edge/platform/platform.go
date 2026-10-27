// Package platform provides operating system abstraction helpers for PATH resolution and directories.
package platform

// LookPath searches for an executable named file in directories named by PATH.
func LookPath(file string) (string, error) {
	return lookPath(file)
}

// ConfigDir returns the user configuration directory for soundcheck.
func ConfigDir() (string, error) {
	return configDir()
}

// CacheDir returns the user cache directory for soundcheck.
func CacheDir() (string, error) {
	return cacheDir()
}

// EnrichEnv ensures that PATH in the environment slice includes existing candidate CLI directories.
func EnrichEnv(env []string) []string {
	return enrichEnv(env)
}
