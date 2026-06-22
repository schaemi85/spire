package service

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// ServiceCmd is the parent for all `spire service` subcommands.
var ServiceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage services in the current Spire project",
}

func init() {
	ServiceCmd.AddCommand(AddCmd)
}

// --- Service config helpers (used by upgrade hooks) ---

type ServiceMeta struct {
	Name    string
	WithDB  bool
	WithAPI bool
	WithJob bool
}

// IsValidServiceDir checks if a directory entry is a valid service directory.
func IsValidServiceDir(servicesDir string, entry os.DirEntry) bool {
	if !entry.IsDir() {
		return false
	}
	_, err := os.Stat(filepath.Join(servicesDir, entry.Name(), "go.mod"))
	return err == nil
}

// ReadServiceConfig reads and parses the config.yaml for a single service.
func ReadServiceConfig(servicePath string) *ServiceMeta {
	b, err := os.ReadFile(filepath.Join(servicePath, "config.yaml"))
	if err != nil {
		return nil
	}
	var rawConfig map[string]interface{}
	if err := yaml.Unmarshal(b, &rawConfig); err != nil {
		return nil
	}
	_, hasDB := rawConfig["db"]
	_, hasAPI := rawConfig["api"]
	_, hasJob := rawConfig["configure_as_job"]
	return &ServiceMeta{
		Name:    filepath.Base(servicePath),
		WithDB:  hasDB,
		WithAPI: hasAPI,
		WithJob: hasJob,
	}
}

func ReadServicesConfigs() ([]ServiceMeta, error) {
	servicesDir := "services"
	entries, err := os.ReadDir(servicesDir)
	if err != nil {
		return nil, fmt.Errorf("error reading services directory: %v", err)
	}
	var services []ServiceMeta
	for _, entry := range entries {
		if !IsValidServiceDir(servicesDir, entry) {
			continue
		}
		sPath := filepath.Join(servicesDir, entry.Name())
		if svc := ReadServiceConfig(sPath); svc != nil {
			services = append(services, *svc)
		}
	}
	return services, nil
}
