// Copyright 2023 Specter Ops, Inc.
// 
// Licensed under the Apache License, Version 2.0
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// 
//     http://www.apache.org/licenses/LICENSE-2.0
// 
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// 
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/specterops/bloodhound/crypto"
	"github.com/specterops/bloodhound/log"
	"github.com/specterops/bloodhound/src/serde"
)

const (
	CurrentConfigurationVersion = 2
	DefaultLogFilePath          = "/var/log/bhapi.log"

	bhAPIEnvironmentVariablePrefix       = "bhe"
	environmentVariablePathSeparator     = "_"
	environmentVariableKeyValueSeparator = "="
)

type TLSConfiguration struct {
	CertFile string `json:"cert_file"`
	KeyFile  string `json:"key_file"`
}

func (s TLSConfiguration) Enabled() bool {
	return s.CertFile != "" && s.KeyFile != ""
}

type DatabaseConfiguration struct {
	Connection            string `json:"connection"`
	Address               string `json:"addr"`
	Database              string `json:"database"`
	Username              string `json:"username"`
	Secret                string `json:"secret"`
	MaxConcurrentSessions int    `json:"max_concurrent_sessions"`
}

type CollectorManifest struct {
	Latest   string             `json:"latest"`
	Versions []CollectorVersion `json:"versions"`
}

type CollectorVersion struct {
	Version    string `json:"version"`
	SHA256Sum  string `json:"sha256sum"`
	Deprecated bool   `json:"deprecated"`
}

type CollectorManifests map[string]CollectorManifest

func (s DatabaseConfiguration) PostgreSQLConnectionString() string {
	if s.Connection == "" {
		return fmt.Sprintf("postgresql://%s:%s@%s/%s", s.Username, s.Secret, s.Address, s.Database)
	}

	return s.Connection
}

func (s DatabaseConfiguration) Neo4jConnectionString() string {
	if s.Connection == "" {
		return fmt.Sprintf("neo4j://%s:%s@%s/%s", s.Username, s.Secret, s.Address, s.Database)
	}

	return s.Connection
}

type CryptoConfiguration struct {
	JWT    JWTConfiguration    `json:"jwt"`
	Argon2 Argon2Configuration `json:"argon2"`
}

type JWTConfiguration struct {
	SigningKey string `json:"signing_key"`
}

func (s *JWTConfiguration) SetSigningKeyBytes(signingKeyBytes []byte) {
	s.SigningKey = base64.StdEncoding.EncodeToString(signingKeyBytes)
}

func (s JWTConfiguration) SigningKeyBytes() ([]byte, error) {
	return base64.StdEncoding.DecodeString(s.SigningKey)
}

type Argon2Configuration struct {
	MemoryKibibytes uint32 `json:"memory_kibibytes"`
	NumIterations   uint32 `json:"num_iterations"`
	NumThreads      uint8  `json:"num_threads"`
}

func (s Argon2Configuration) NewDigester() crypto.SecretDigester {
	return crypto.Argon2{
		MemoryKibibytes: s.MemoryKibibytes,
		NumIterations:   s.NumIterations,
		NumThreads:      s.NumThreads,
	}
}

type SAMLConfiguration struct {
	ServiceProviderCertificate        string `json:"sp_cert"`
	ServiceProviderKey                string `json:"sp_key"`
	ServiceProviderCertificateCAChain string `json:"sp_ca_chain"`
}

type SpecterAuthConfiguration struct {
	InstanceUUID string `json:"instance_uuid"`
	Token        string `json:"token"`
}

type DefaultAdminConfiguration struct {
	PrincipalName string `json:"principal_name"`
	Password      string `json:"password"`
	EmailAddress  string `json:"email_address"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	ExpireNow     bool   `json:"expire_now"`
}

type Configuration struct {
	Version                int                       `json:"version"`
	BindAddress            string                    `json:"bind_addr"`
	NetTimeoutSeconds      int                       `json:"net_timeout_seconds"`
	SlowQueryThreshold     int64                     `json:"slow_query_threshold"`
	MaxGraphQueryCacheSize int                       `json:"max_graphdb_cache_size"`
	MaxAPICacheSize        int                       `json:"max_api_cache_size"`
	MetricsPort            string                    `json:"metrics_port"`
	RootURL                serde.URL                 `json:"root_url"`
	WorkDir                string                    `json:"work_dir"`
	LogLevel               string                    `json:"log_level"`
	LogPath                string                    `json:"log_path"`
	TLS                    TLSConfiguration          `json:"tls"`
	Database               DatabaseConfiguration     `json:"database"`
	Neo4J                  DatabaseConfiguration     `json:"neo4j"`
	Crypto                 CryptoConfiguration       `json:"crypto"`
	SAML                   SAMLConfiguration         `json:"saml"`
	SpecterAuth            SpecterAuthConfiguration  `json:"specter_auth"`
	DefaultAdmin           DefaultAdminConfiguration `json:"default_admin"`
	CollectorsBasePath     string                    `json:"collectors_base_path"`
	DatapipeInterval       int                       `json:"datapipe_interval"`
	EnableAPILogging       bool                      `json:"enable_api_logging"`
	DisableEnrichment      bool                      `json:"disable_enrichment"`
	DisableCypherQC        bool                      `json:"disable_cypher_qc"`
}

func (s Configuration) TempDirectory() string {
	return filepath.Join(s.WorkDir, "tmp")
}

func (s Configuration) ClientLogDirectory() string {
	return filepath.Join(s.WorkDir, "client_logs")
}

func (s Configuration) CollectorsDirectory() string {
	return s.CollectorsBasePath
}

func WriteConfigurationFile(path string, config Configuration) error {
	if fout, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644); err != nil {
		return fmt.Errorf("failed opening configuration file %s: %w", path, err)
	} else {
		defer fout.Close()

		if content, err := json.MarshalIndent(config, "", "    "); err != nil {
			return fmt.Errorf("failed serializing configuration to json: %w", err)
		} else if _, err := fout.Write(content); err != nil {
			return fmt.Errorf("failed writing to confniguration to file %s: %w", path, err)
		}
	}

	return nil
}

func ParseConfiguration(content []byte) (Configuration, error) {
	if configuration, err := NewDefaultConfiguration(); err != nil {
		return configuration, fmt.Errorf("failed to create default configuration: %w", err)
	} else {
		return configuration, json.Unmarshal(content, &configuration)
	}
}

func ReadConfigurationFile(path string) (Configuration, error) {
	if content, err := os.ReadFile(path); err != nil {
		return Configuration{}, err
	} else {
		return ParseConfiguration(content)
	}
}

func HasConfigurationFile(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func formatEnvironmentVariablePrefix(varPrefix string) string {
	if !strings.HasSuffix(varPrefix, environmentVariablePathSeparator) {
		return varPrefix + environmentVariablePathSeparator
	}

	return varPrefix
}

func SetValuesFromEnv(varPrefix string, target any, env []string) error {
	for _, kvPairStr := range env {
		if kvParts := strings.SplitN(kvPairStr, environmentVariableKeyValueSeparator, 2); len(kvParts) == 2 {
			var (
				key      = strings.TrimSpace(kvParts[0])
				valueStr = strings.TrimSpace(kvParts[1])
			)

			if formattedPrefix := formatEnvironmentVariablePrefix(varPrefix); strings.HasPrefix(key, formattedPrefix) {
				cfgKeyPath := strings.TrimPrefix(key, formattedPrefix)

				if err := SetValue(target, cfgKeyPath, valueStr); err != nil {
					return err
				}
			}
		} else {
			log.Errorf("Invalid key/value pair: %+v", kvParts)
		}
	}

	return nil
}

func GetConfiguration(path string) (Configuration, error) {
	cfg, err := NewDefaultConfiguration()
	if err != nil {
		return cfg, fmt.Errorf("failed to create default configuration: %w", err)
	}

	if hasCfgFile, err := HasConfigurationFile(path); err != nil {
		return Configuration{}, err
	} else if hasCfgFile {
		log.Infof("Reading configuration found at %s", path)

		if readCfg, err := ReadConfigurationFile(path); err != nil {
			return Configuration{}, err
		} else {
			cfg = readCfg
		}
	} else {
		log.Infof("No configuration file found at %s", path)
	}

	if err := SetValuesFromEnv(bhAPIEnvironmentVariablePrefix, &cfg, os.Environ()); err != nil {
		return Configuration{}, err
	}

	return cfg, nil
}

func (s Configuration) SaveCollectorManifests() (CollectorManifests, error) {
	if azureHoundManifest, err := generateCollectorManifest(filepath.Join(s.CollectorsDirectory(), "azurehound")); err != nil {
		return CollectorManifests{}, fmt.Errorf("error generating AzureHound manifest file: %w", err)
	} else if sharpHoundManifest, err := generateCollectorManifest(filepath.Join(s.CollectorsDirectory(), "sharphound")); err != nil {
		return CollectorManifests{}, fmt.Errorf("error generating SharpHound manifest file: %w", err)
	} else {
		return CollectorManifests{
			"azurehound": azureHoundManifest,
			"sharphound": sharpHoundManifest,
		}, nil
	}
}

func generateCollectorManifest(collectorDir string) (CollectorManifest, error) {
	var (
		collectorVersions []CollectorVersion
		latestVersion     string
	)

	if semverRegex, err := regexp.Compile("v[0-9]+.[0-9]+.[0-9]+"); err != nil {
		return CollectorManifest{}, fmt.Errorf("error compiling semver regex: %w", err)
	} else if collectorFiles, err := os.ReadDir(collectorDir); err != nil {
		return CollectorManifest{}, fmt.Errorf("error reading downloads directory %s: %w", collectorDir, err)
	} else {
		for _, collectorFile := range collectorFiles {
			name := collectorFile.Name()
			if filepath.Ext(name) == ".zip" {
				if version := semverRegex.Find([]byte(name)); version == nil {
					continue
				} else if sha256, err := os.ReadFile(filepath.Join(collectorDir, name+".sha256")); err != nil {
					return CollectorManifest{}, fmt.Errorf("error reading sha256 file for %s: %w", name, err)
				} else {
					collectorVersions = append(collectorVersions, CollectorVersion{
						Version:    string(version),
						SHA256Sum:  strings.Fields(string(sha256))[0], // Get only the SHA-256 portion
						Deprecated: strings.Contains(collectorDir, "sharphound") && string(version) < "v2.0.0",
					})

					if string(version) > latestVersion {
						latestVersion = string(version)
					}
				}
			}
		}
	}

	return CollectorManifest{
		Latest:   latestVersion,
		Versions: collectorVersions,
	}, nil
}
