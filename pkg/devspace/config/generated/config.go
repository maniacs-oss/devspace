package generated

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	yaml "gopkg.in/yaml.v2"
)

// Config specifies the runtime config struct
type Config struct {
	OverrideProfile *string                 `yaml:"lastOverrideProfile,omitempty"`
	ActiveProfile   string                  `yaml:"activeProfile,omitempty"`
	Vars            map[string]string       `yaml:"vars,omitempty"`
	Profiles        map[string]*CacheConfig `yaml:"profiles,omitempty"`
}

// LastContextConfig holds all the informations about the last used kubernetes context
type LastContextConfig struct {
	Namespace string `yaml:"namespace,omitempty"`
	Context   string `yaml:"context,omitempty"`
}

// CacheConfig holds all the information specific to a certain config
type CacheConfig struct {
	Deployments  map[string]*DeploymentCache `yaml:"deployments,omitempty"`
	Images       map[string]*ImageCache      `yaml:"images,omitempty"`
	Dependencies map[string]string           `yaml:"dependencies,omitempty"`
	LastContext  *LastContextConfig          `yaml:"lastContext,omitempty"`
}

// ImageCache holds the cache related information about a certain image
type ImageCache struct {
	ImageConfigHash string `yaml:"imageConfigHash,omitempty"`

	DockerfileHash string `yaml:"dockerfileHash,omitempty"`
	ContextHash    string `yaml:"contextHash,omitempty"`
	EntrypointHash string `yaml:"entrypointHash,omitempty"`

	CustomFilesHash string `yaml:"customFilesHash,omitempty"`

	ImageName string `yaml:"imageName,omitempty"`
	Tag       string `yaml:"tag,omitempty"`
}

// DeploymentCache holds the information about a specific deployment
type DeploymentCache struct {
	DeploymentConfigHash string `yaml:"deploymentConfigHash,omitempty"`

	HelmOverridesHash    string `yaml:"helmOverridesHash,omitempty"`
	HelmChartHash        string `yaml:"helmChartHash,omitempty"`
	KubectlManifestsHash string `yaml:"kubectlManifestsHash,omitempty"`
}

// ConfigPath is the relative generated config path
var ConfigPath = ".devspace/generated.yaml"

var loadedConfig *Config
var loadedConfigOnce sync.Once

var testDontSaveConfig = false

// SetTestConfig sets the config for testing purposes
func SetTestConfig(config *Config) {
	loadedConfigOnce.Do(func() {})
	loadedConfig = config
	testDontSaveConfig = true
}

//ResetConfig resets the config to nil and enables loading from configs.yaml
func ResetConfig() {
	loadedConfigOnce = sync.Once{}
	loadedConfig = nil
}

// LoadConfig loads the config from the filesystem
func LoadConfig(ctx context.Context) (*Config, error) {
	var err error

	loadedConfigOnce.Do(func() {
		loadedConfig, err = LoadConfigFromPath(ctx, ConfigPath)
	})

	return loadedConfig, err
}

// LoadConfigFromPath loads the generated config from a given path
func LoadConfigFromPath(ctx context.Context, path string) (*Config, error) {
	var loadedConfig *Config

	data, readErr := ioutil.ReadFile(path)
	if readErr != nil {
		loadedConfig = &Config{
			OverrideProfile: nil,
			ActiveProfile:   "",
			Profiles:        make(map[string]*CacheConfig),
			Vars:            make(map[string]string),
		}
	} else {
		loadedConfig = &Config{}
		err := yaml.Unmarshal(data, loadedConfig)
		if err != nil {
			return nil, err
		}

		if loadedConfig.Profiles == nil {
			loadedConfig.Profiles = make(map[string]*CacheConfig)
		}
		if loadedConfig.Vars == nil {
			loadedConfig.Vars = make(map[string]string)
		}
	}

	// Set override profile
	if ctx.Value(constants.ProfileContextKey) != nil && ctx.Value(constants.ProfileContextKey).(string) != "" {
		loadedConfig.OverrideProfile = ptr.String(ctx.Value(constants.ProfileContextKey).(string))
	} else {
		loadedConfig.OverrideProfile = nil
	}

	InitDevSpaceConfig(loadedConfig, loadedConfig.ActiveProfile)
	return loadedConfig, nil
}

// NewCache returns a new cache object
func NewCache() *CacheConfig {
	return &CacheConfig{
		Deployments: make(map[string]*DeploymentCache),
		Images:      make(map[string]*ImageCache),

		Dependencies: make(map[string]string),
	}
}

// GetActive returns the currently active devspace config
func (config *Config) GetActive() *CacheConfig {
	active := config.ActiveProfile
	if config.OverrideProfile != nil {
		active = *config.OverrideProfile
	}

	InitDevSpaceConfig(config, active)
	return config.Profiles[active]
}

// GetImageCache returns the image cache if it exists and creates one if not
func (cache *CacheConfig) GetImageCache(imageConfigName string) *ImageCache {
	if _, ok := cache.Images[imageConfigName]; !ok {
		cache.Images[imageConfigName] = &ImageCache{}
	}

	return cache.Images[imageConfigName]
}

// GetDeploymentCache returns the deployment cache if it exists and creates one if not
func (cache *CacheConfig) GetDeploymentCache(deploymentName string) *DeploymentCache {
	if _, ok := cache.Deployments[deploymentName]; !ok {
		cache.Deployments[deploymentName] = &DeploymentCache{}
	}

	return cache.Deployments[deploymentName]
}

// InitDevSpaceConfig verifies a given config name is set
func InitDevSpaceConfig(config *Config, configName string) {
	if _, ok := config.Profiles[configName]; ok == false {
		config.Profiles[configName] = NewCache()
		return
	}

	if config.Profiles[configName].Deployments == nil {
		config.Profiles[configName].Deployments = make(map[string]*DeploymentCache)
	}
	if config.Profiles[configName].Images == nil {
		config.Profiles[configName].Images = make(map[string]*ImageCache)
	}
	if config.Profiles[configName].Dependencies == nil {
		config.Profiles[configName].Dependencies = make(map[string]string)
	}
}

// SaveConfig saves the config to the filesystem
func SaveConfig(config *Config) error {
	if testDontSaveConfig {
		return nil
	}

	workdir, _ := os.Getwd()
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	InitDevSpaceConfig(config, config.ActiveProfile)

	configPath := filepath.Join(workdir, ConfigPath)
	err = os.MkdirAll(filepath.Dir(configPath), 0755)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(configPath, data, 0666)
}
