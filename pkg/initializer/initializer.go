package initializer

import (
	"encoding/base64"
	"fmt"
	"github.com/DevopsArtFactory/goployer/pkg/builder"
	"github.com/ghodss/yaml"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/DevopsArtFactory/goployer/pkg/tool"
	Logger "github.com/sirupsen/logrus"
)

var (
	manifestDir      = "manifests"
	scriptDir        = "scripts"
	scriptPath       = fmt.Sprintf("%s/%s", scriptDir, "userdata.sh")
	metricPath       = "metrics.yaml"
	manifestBase64   = "LS0tCm5hbWU6IGhlbGxvCnVzZXJkYXRhOgogIHR5cGU6IGxvY2FsCiAgcGF0aDogZXhhbXBsZXMvc2NyaXB0cy91c2VyZGF0YS5zaAoKYXV0b3NjYWxpbmc6ICZhdXRvc2NhbGluZ19wb2xpY3kKICAtIG5hbWU6IHNjYWxlX2luCiAgICBhZGp1c3RtZW50X3R5cGU6IENoYW5nZUluQ2FwYWNpdHkKICAgIHNjYWxpbmdfYWRqdXN0bWVudDogLTEKICAgIGNvb2xkb3duOiA2MAogIC0gbmFtZTogc2NhbGVfb3V0CiAgICBhZGp1c3RtZW50X3R5cGU6IENoYW5nZUluQ2FwYWNpdHkKICAgIHNjYWxpbmdfYWRqdXN0bWVudDogMQogICAgY29vbGRvd246IDE4MAoKYWxhcm1zOiAmYXV0b3NjYWxpbmdfYWxhcm1zCiAgLSBuYW1lOiBzY2FsZV9vdXRfb25fdXRpbAogICAgbmFtZXNwYWNlOiBBV1MvRUMyCiAgICBtZXRyaWM6IENQVVV0aWxpemF0aW9uCiAgICBzdGF0aXN0aWM6IEF2ZXJhZ2UKICAgIGNvbXBhcmlzb246IEdyZWF0ZXJUaGFuT3JFcXVhbFRvVGhyZXNob2xkCiAgICB0aHJlc2hvbGQ6IDUwCiAgICBwZXJpb2Q6IDEyMAogICAgZXZhbHVhdGlvbl9wZXJpb2RzOiAyCiAgICBhbGFybV9hY3Rpb25zOgogICAgICAtIHNjYWxlX291dAogIC0gbmFtZTogc2NhbGVfaW5fb25fdXRpbAogICAgbmFtZXNwYWNlOiBBV1MvRUMyCiAgICBtZXRyaWM6IENQVVV0aWxpemF0aW9uCiAgICBzdGF0aXN0aWM6IEF2ZXJhZ2UKICAgIGNvbXBhcmlzb246IExlc3NUaGFuT3JFcXVhbFRvVGhyZXNob2xkCiAgICB0aHJlc2hvbGQ6IDMwCiAgICBwZXJpb2Q6IDMwMAogICAgZXZhbHVhdGlvbl9wZXJpb2RzOiAzCiAgICBhbGFybV9hY3Rpb25zOgogICAgICAtIHNjYWxlX2luCgojIFRhZ3Mgc2hvdWxkIGJlIGxpa2UgImtleT12YWx1ZSIKdGFnczoKICAtIHByb2plY3Q9dGVzdAogIC0gcmVwbz1oZWxsby1kZXBsb3kKCnN0YWNrczoKICAtIHN0YWNrOiBhcnRkCgogICAgcG9sbGluZ19pbnRlcnZhbDogMzBzCgogICAgIyBhY2NvdW50IGFsaWFzCiAgICBhY2NvdW50OiBkZXYKCiAgICAjIGVudmlyb25tZW50IHZhcmlhYmxlCiAgICBlbnY6IGRldgoKICAgICMgYXNzdW1lX3JvbGUgZm9yIGRlcGxveW1lbnQKICAgIGFzc3VtZV9yb2xlOiAiIgoKICAgICMgUmVwbGFjZW1lbnQgdHlwZQogICAgcmVwbGFjZW1lbnRfdHlwZTogQmx1ZUdyZWVuCgogICAgIyBJQU0gaW5zdGFuY2UgcHJvZmlsZSwgbm90IElBTSByb2xlCiAgICBpYW1faW5zdGFuY2VfcHJvZmlsZTogJ2FwcC1oZWxsby1wcm9maWxlJwoKICAgICMgU3RhY2sgc3BlY2lmaWMgdGFncwogICAgYW5zaWJsZV90YWdzOiBhbGwKICAgIHRhZ3M6CiAgICAgIC0gc3RhY2stbmFtZT1hcnRkCiAgICAgIC0gdGVzdD10ZXN0CgogICAgIyBFQlMgT3B0aW1pemVkCiAgICBlYnNfb3B0aW1pemVkOiB0cnVlCgogICAgIyBpbnN0YW5jZV9tYXJrZXRfb3B0aW9ucyBpcyBmb3Igc3BvdCB1c2FnZQogICAgIyBZb3Ugb25seSBjYW4gY2hvb3NlIHNwb3QgYXMgbWFya2V0X3R5cGUuCiAgICAjIElmIHlvdSB3YW50IHRvIHNldCBjdXN0b21pemVkIHN0b3Agb3B0aW9ucywgdGhlbiBwbGVhc2Ugd3JpdGUgc3BvdF9vcHRpb25zIGNvcnJlY3RseS4KICAgICNpbnN0YW5jZV9tYXJrZXRfb3B0aW9uczoKICAgICMgIG1hcmtldF90eXBlOiBzcG90CiAgICAjICBzcG90X29wdGlvbnM6CiAgICAjICAgIGJsb2NrX2R1cmF0aW9uX21pbnV0ZXM6IDE4MAogICAgIyAgICBpbnN0YW5jZV9pbnRlcnJ1cHRpb25fYmVoYXZpb3I6IHRlcm1pbmF0ZSAjIHRlcm1pbmF0ZSAvIHN0b3AgLyBoaWJlcm5hdGUKICAgICMgICAgbWF4X3ByaWNlOiAwLgogICAgIyAgICBzcG90X2luc3RhbmNlX3R5cGU6IG9uZS10aW1lICMgb25lLXRpbWUgb3IgcGVyc2lzdGVudAoKCiAgICAjIE1peGVkSW5zdGFuY2VzUG9saWN5CiAgICAjIFlvdSBjYW4gc2V0IGF1dG9zY2FsaW5nIG1peGVkSW5zdGFuY2VQb2xpY3kgdG8gdXNlIG9uIGRlbWFuZCBhbmQgc3BvdCBpbnN0YW5jZXMgdG9nZXRoZXIuCiAgICAjIGlmIG1peGVkX2luc3RhbmNlX3BvbGljeSBpcyBzZXQsIHRoZW4gYGluc3RhbmNlX21hcmtldF9vcHRpb25zYCB3aWxsIGJlIGlnbm9yZWQuCiAgICBtaXhlZF9pbnN0YW5jZXNfcG9saWN5OgogICAgICBlbmFibGVkOiBmYWxzZQoKICAgICAgIyBpbnN0YW5jZSB0eXBlIGxpc3QgdG8gb3ZlcnJpZGUgdGhlIGluc3RhbmNlIHR5cGVzIGluIGxhdW5jaCB0ZW1wbGF0ZS4KICAgICAgb3ZlcnJpZGVfaW5zdGFuY2VfdHlwZXM6CiAgICAgICAgLSBjNS5sYXJnZQogICAgICAgIC0gYzUueGxhcmdlCgogICAgICAjIFByb3BvcnRpb24gb2Ygb24tZGVtYW5kIGluc3RhbmNlcy4KICAgICAgIyBCeSBkZWZhdWx0LCB0aGlzIHZhbHVlICB3aWxsIGJlIDEwMCB3aGljaCBtZWFucyBubyBzcG90IGluc3RhbmNlLgogICAgICBvbl9kZW1hbmRfcGVyY2VudGFnZTogMjAKCiAgICAgICMgc3BvdF9hbGxvY2F0aW9uX3N0cmF0ZWd5IG1lYW5zIGluIHdoYXQgc3RyYXRlZ3kgeW91IHdhbnQgdG8gYWxsb2NhdGUgc3BvdCBpbnN0YW5jZXMuCiAgICAgICMgb3B0aW9ucyBjb3VsZCBiZSBlaXRoZXIgYGxvd2VzdC1wcmljZWAgb3IgYGNhcGFjaXR5LW9wdGltaXplZGAuCiAgICAgICMgYnkgZGVmYXVsdCwgYGxvdy1wcmljZWAgc3RyYXRlZ3kgd2lsbCBiZSBhcHBsaWVkLgogICAgICBzcG90X2FsbG9jYXRpb25fc3RyYXRlZ3k6IGxvd2VzdC1wcmljZQoKICAgICAgIyBUaGUgbnVtYmVyIG9mIHNwb3QgaW5zdGFuY2VzIHBvb2wuCiAgICAgICMgVGhpcyB3aWxsIGJlIHNldCBhbW9uZyBpbnN0YW5jZSB0eXBlcyBpbiBgb3ZlcnJpZGVgIGZpZWxkcwogICAgICAjIFRoaXMgd2lsbCBiZSB2YWxpZCBvbmx5IGlmIHRoZSBgc3BvdF9hbGxvY2F0aW9uX3N0cmF0ZWd5YCBpcyBsb3ctcHJpY2UuCiAgICAgIHNwb3RfaW5zdGFuY2VfcG9vbHM6IDMKCiAgICAgICMgU3BvdCBwcmljZS4KICAgICAgIyBCeSBkZWZhdWx0LCBvbi1kZW1hbmQgcHJpY2Ugd2lsbCBiZSBhdXRvbWF0aWNhbGx5IGFwcGxpZWQuCiAgICAgIHNwb3RfbWF4X3ByaWNlOiAwLjMKCiAgICAjIGJsb2NrX2RldmljZXMgaXMgdGhlIGxpc3Qgb2YgZWJzIHZvbHVtZXMgeW91IGNhbiB1c2UgZm9yIGVjMgogICAgIyBkZXZpY2VfbmFtZSBpcyByZXF1aXJlZAogICAgIyBJZiB5b3UgZG8gbm90IHNldCB2b2x1bWVfc2l6ZSwgaXQgd291bGQgYmUgMTYuCiAgICAjIElmIHlvdSBkbyBub3Qgc2V0IHZvbHVtZV90eXBlLCBpdCB3b3VsZCBiZSBncDIuCiAgICBibG9ja19kZXZpY2VzOgogICAgICAtIGRldmljZV9uYW1lOiAvZGV2L3h2ZGEKICAgICAgICB2b2x1bWVfc2l6ZTogMTAwCiAgICAgICAgdm9sdW1lX3R5cGU6ICJncDIiCiAgICAgIC0gZGV2aWNlX25hbWU6IC9kZXYveHZkYgogICAgICAgIHZvbHVtZV90eXBlOiAic3QxIgogICAgICAgIHZvbHVtZV9zaXplOiA1MDAKCiAgICAjIGNhcGFjaXR5CiAgICBjYXBhY2l0eToKICAgICAgbWluOiAxCiAgICAgIG1heDogMgogICAgICBkZXNpcmVkOiAxCgogICAgIyBhdXRvc2NhbGluZyBtZWFucyBzY2FsaW5nIHBvbGljeSBvZiBhdXRvc2NhbGluZyBncm91cAogICAgIyBZb3UgY2FuIGZpbmQgZm9ybWF0IGluIGF1dG9zY2FsaW5nIGJsb2NrIHVwc2lkZQogICAgYXV0b3NjYWxpbmc6ICphdXRvc2NhbGluZ19wb2xpY3kKCiAgICAjIGFsYXJtcyBtZWFucyBjbG91ZHdhdGNoIGFsYXJtcyBmb3IgdHJpZ2dlcmluZyBhdXRvc2NhbGluZyBzY2FsaW5nIHBvbGljeQogICAgIyBZb3UgY2FuIGZpbmQgZm9ybWF0IGluIGFsYXJtcyBibG9jayB1cHNpZGUKICAgIGFsYXJtczogKmF1dG9zY2FsaW5nX2FsYXJtcwoKICAgICMgbGlmZWN5Y2xlIGNhbGxiYWNrcwogICAgbGlmZWN5Y2xlX2NhbGxiYWNrczoKICAgICAgcHJlX3Rlcm1pbmF0ZV9wYXN0X2NsdXN0ZXI6CiAgICAgICAgLSBzZXJ2aWNlIGhlbGxvIHN0b3AKCiAgICAjIGxpc3Qgb2YgcmVnaW9uCiAgICAjIGRlcGxveWVyIHdpbGwgY29uY3VycmVudGx5IGRlcGxveSBhY3Jvc3MgdGhlIHJlZ2lvbgogICAgcmVnaW9uczoKICAgICAgLSByZWdpb246IGFwLW5vcnRoZWFzdC0yCgogICAgICAgICMgaW5zdGFuY2UgdHlwZQogICAgICAgIGluc3RhbmNlX3R5cGU6IHQzLm1lZGl1bQoKICAgICAgICAjIHNzaF9rZXkgZm9yIGluc3RhbmNlcwogICAgICAgIHNzaF9rZXk6IHRlc3QtbWFzdGVyLWtleQoKICAgICAgICAjIGFtaV9pZAogICAgICAgICMgWW91IGNhbiBvdmVycmlkZSB0aGlzIHZhbHVlIHZpYSBjb21tYW5kIGxpbmUgYC0tYW1pYAogICAgICAgIGFtaV9pZDogYW1pLTAxMjg4OTQ1YmQyNGVkNDlhCgogICAgICAgICMgV2hldGhlciB5b3Ugd2FudCB0byB1c2UgcHVibGljIHN1Ym5ldCBvciBub3QKICAgICAgICAjIEJ5IERlZmF1bHQsIGRlcGxveWVyIHNlbGVjdHMgcHJpdmF0ZSBzdWJuZXRzCiAgICAgICAgIyBJZiB5b3Ugd2FudCB0byB1c2UgcHVibGljIHN1Ym5ldCwgdGhlbiB5b3Ugc2hvdWxkIHNldCB0aGlzIHZhbHVlIHRvIHR1cmUuCiAgICAgICAgdXNlX3B1YmxpY19zdWJuZXRzOiB0cnVlCgogICAgICAgICMgWW91IGNhbiB1c2UgVlBDIGlkKHZwYy14eHgpCiAgICAgICAgIyBJZiB5b3Ugc3BlY2lmeSB0aGUgbmFtZSBvZiBWUEMsIHRoZW4gZGVwbG95ZXIgd2lsbCBmaW5kIHRoZSBWUEMgaWQgd2l0aCBpdC4KICAgICAgICAjIEluIHRoaXMgY2FzZSwgb25seSBvbmUgVlBDIHNob3VsZCBleGlzdC4KICAgICAgICB2cGM6IHZwYy1hcnRkX2Fwbm9ydGhlYXN0MgoKICAgICAgICAjIFlvdSBjYW4gdXNlIHNlY3VyaXR5IGdyb3VwIGlkKHNnLXh4eCkKICAgICAgICAjIElmIHlvdSBzcGVjaWZ5IHRoZSBuYW1lIG9mIHNlY3VyaXR5IGdyb3VwLCB0aGVuIGRlcGxveWVyIHdpbGwgZmluZCB0aGUgc2VjdXJpdHkgZ3JvdXAgaWQgd2l0aCBpdC4KICAgICAgICAjIEluIHRoaXMgY2FzZSwgb25seSBvbmUgc2VjdXJpdHkgZ3JvdXAgc2hvdWxkIGV4aXN0CiAgICAgICAgc2VjdXJpdHlfZ3JvdXBzOgogICAgICAgICAgLSBoZWxsby1hcnRkX2Fwbm9ydGhlYXN0MgogICAgICAgICAgLSBkZWZhdWx0LWFydGRfYXBub3J0aGVhc3QyCgogICAgICAgICMgWW91IGNhbiB1c2UgaGVhbHRoY2hlY2sgdGFyZ2V0IGdyb3VwCiAgICAgICAgaGVhbHRoY2hlY2tfdGFyZ2V0X2dyb3VwOiBoZWxsby1hcnRkYXBuZTItZXh0CgogICAgICAgICMgSWYgbm8gYXZhaWxhYmlsaXR5IHpvbmVzIHNwZWNpZmllZCwgdGhlbiBhbGwgYXZhaWxhYmlsaXR5IHpvbmVzIGFyZSBzZWxlY3RlZCBieSBkZWZhdWx0LgogICAgICAgICMgSWYgeW91IHdhbnQgYWxsIGF2YWlsYWJpbGl0eSB6b25lcywgdGhlbiBwbGVhc2UgcmVtb3ZlIGF2YWlsYWJpbGl0eV96b25lcyBrZXkuCiAgICAgICAgYXZhaWxhYmlsaXR5X3pvbmVzOgogICAgICAgICAgLSBhcC1ub3J0aGVhc3QtMmEKICAgICAgICAgIC0gYXAtbm9ydGhlYXN0LTJiCiAgICAgICAgICAtIGFwLW5vcnRoZWFzdC0yYwoKICAgICAgICAjIGxpc3Qgb2YgdGFyZ2V0IGdyb3Vwcy4KICAgICAgICAjIFRoZSB0YXJnZXQgZ3JvdXAgaW4gdGhlIGhlYWx0aGNoZWNrX3RhcmdldF9ncm91cCBzaG91bGQgYmUgaW5jbHVkZWQgaGVyZS4KICAgICAgICB0YXJnZXRfZ3JvdXBzOgogICAgICAgICAgLSBoZWxsby1hcnRkYXBuZTItZXh0Cg=="
	userdataBase64   = "IyEvYmluL2Jhc2gKCnN1ZG8geXVtIHVwZGF0ZQpzdWRvIGFtYXpvbi1saW51eC1leHRyYXMgaW5zdGFsbCBuZ2lueDEuMTIKc3VkbyBzZXJ2aWNlIG5naW54IHN0YXJ0"
	metricFileBase64 = "cmVnaW9uOiBhcC1ub3J0aGVhc3QtMgpzdG9yYWdlOgogIHR5cGU6IGR5bmFtb2RiCiAgbmFtZTogZ29wbG95ZXItbWV0cmljcwoK"
)

type Initializer struct {
	AppName    string
	Logger     *Logger.Logger
	YamlConfig builder.YamlConfig
}

func NewInitializer(appName string) Initializer {
	return Initializer{
		AppName: appName,
		Logger:  Logger.New(),
		YamlConfig: builder.YamlConfig{
			Name:     appName,
			Userdata: builder.Userdata{},
			Tags:     []string{},
			Stacks:   []builder.Stack{},
		},
	}
}

// RunInit creates necessary files
func (i Initializer) RunInit() error {
	filePath := fmt.Sprintf("%s/%s.yaml", manifestDir, i.AppName)

	// Generate data
	writeData, err := generateData(filePath, manifestBase64)
	if err != nil {
		return err
	}

	scriptData, err := generateData(scriptPath, userdataBase64)
	if err != nil {
		return err
	}

	metricData, err := generateData(metricPath, metricFileBase64)
	if err != nil {
		return err
	}

	if !tool.AskContinue("Do you want to add this manifest file? ") {
		tool.Red.Fprintln(os.Stdout, "canceled")
		return nil
	}

	if err := i.CheckDir(filePath); err != nil {
		return err
	}

	i.Logger.Debugf("starts to write yaml configuration: %s", filePath)
	if err := generateFile(filePath, string(writeData)); err != nil {
		return err
	}

	i.Logger.Debugf("starts to write script data: %s", scriptPath)
	if err := generateFile(scriptPath, scriptData); err != nil {
		return err
	}

	i.Logger.Debugf("starts to write metric configuration: %s", metricPath)
	if err := generateFile(metricPath, metricData); err != nil {
		return err
	}

	fmt.Println("files are successfully created")
	tool.Blue.Fprintln(os.Stdout, "Change the value of configurations based on your environment before using")
	return nil
}

func (i Initializer) CheckDir(filePath string) error {
	// check manifest directory
	i.Logger.Debugf("check if manifest directory exists")
	if !tool.CheckFileExists(manifestDir) {
		i.Logger.Debugf("%s directory does not exist!", manifestDir)
		if err := os.Mkdir(manifestDir, os.ModePerm); err != nil {
			return err
		}
		i.Logger.Debugf("%s directory is successfully created!", manifestDir)
	}

	i.Logger.Debugf("file will be added to %s", filePath)
	if !tool.CheckFileExists(filePath) {
		i.Logger.Debugf("file does not exists: %s", filePath)
		f, err := os.Create(filePath)
		if err != nil {
			log.Fatal(err)
		}
		f.Close()
	}

	// check script directory
	i.Logger.Debugf("check if script directory exists")
	if !tool.CheckFileExists(scriptDir) {
		i.Logger.Debugf("%s directory does not exist", scriptDir)
		if err := os.Mkdir(scriptDir, os.ModePerm); err != nil {
			return err
		}
		i.Logger.Debugf("%s directory is successfully created!", scriptDir)
	}

	i.Logger.Debugf("file will be added to %s", scriptPath)
	if !tool.CheckFileExists(scriptPath) {
		i.Logger.Debugf("file does not exists: %s", scriptPath)
		f, err := os.Create(scriptPath)
		if err != nil {
			log.Fatal(err)
		}
		f.Close()
	}

	return nil
}

func (i Initializer) GetWriteData(path string) ([]byte, error) {
	ret, err := yaml.Marshal(i.YamlConfig)
	if err != nil {
		return nil, err
	}

	tool.Yellow.Fprintf(os.Stdout, "%s:\n", path)
	fmt.Println(string(ret))
	fmt.Println()

	return ret, nil
}

func generateData(path, base64String string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		return "", err
	}

	ret := string(data)

	tool.Yellow.Fprintf(os.Stdout, "%s:\n", path)
	fmt.Println(ret)
	fmt.Println()

	return ret, nil
}

func generateFile(filePath string, writeData string) error {
	if err := ioutil.WriteFile(filePath, []byte(writeData), 0644); err != nil {
		return err
	}

	return nil
}

func getYamlConfig(appName string) builder.YamlConfig {
	yc := builder.YamlConfig{}
	yc.Name = appName
	yc.Userdata = builder.Userdata{
		Type: "local",
		Path: "scripts/userdata.sh",
	}

	yc.Tags = []string{fmt.Sprintf("application=%s", appName)}

	yc.Stacks = append(yc.Stacks, builder.Stack{
		Stack:              "artd",
		Account:            "dev",
		Env:                "dev",
		ReplacementType:    "BlueGreen",
		IamInstanceProfile: fmt.Sprintf("app-%s", appName),
		Tags:               []string{"stack-env=dev"},
		PollingInterval:    time.Duration(20 * time.Second),
		EbsOptimized:       true,
		InstanceMarketOptions: &builder.InstanceMarketOptions{
			MarketType: "spot",
			SpotOptions: builder.SpotOptions{
				BlockDurationMinutes:         0,
				InstanceInterruptionBehavior: "terminate",
				MaxPrice:                     "0",
				SpotInstanceType:             "one-time",
			},
		},
		MixedInstancesPolicy: builder.MixedInstancesPolicy{
			Enabled:                false,
			Override:               []string{"c5.large"},
			OnDemandPercentage:     100,
			SpotAllocationStrategy: "lowest-price",
			SpotInstancePools:      1,
			SpotMaxPrice:           "0",
		},
		BlockDevices: []builder.BlockDevice{
			{
				DeviceName: "/dev/xvda",
				VolumeSize: 8,
				VolumeType: "gp2",
			},
		},
		Capacity: builder.Capacity{
			Min:     1,
			Max:     1,
			Desired: 1,
		},
		Autoscaling: []builder.ScalePolicy{
			{
				Name:              "scale_in",
				AdjustmentType:    "ChangeInCapacity",
				ScalingAdjustment: -1,
				Cooldown:          60,
			},
			{
				Name:              "scale_out",
				AdjustmentType:    "ChangeInCapacity",
				ScalingAdjustment: 1,
				Cooldown:          60,
			},
		},
		Alarms: []builder.AlarmConfigs{
			{
				Name:              "scale_out_on_util",
				Namespace:         "AWS/EC2",
				Metric:            "CPUUtilization",
				Statistic:         "Average",
				Comparison:        "GreaterThanOrEqualToThreshold",
				Threshold:         50,
				Period:            120,
				EvaluationPeriods: 2,
				AlarmActions:      []string{"scale_out"},
			},
			{
				Name:              "scale_in_on_util",
				Namespace:         "AWS/EC2",
				Metric:            "CPUUtilization",
				Statistic:         "Average",
				Comparison:        "LessThanOrEqualToThreshold",
				Threshold:         30,
				Period:            300,
				EvaluationPeriods: 2,
				AlarmActions:      []string{"scale_in"},
			},
		},
		LifecycleCallbacks: builder.LifecycleCallbacks{
			PreTerminatePastClusters: []string{
				"echo \"terminate service\"",
			},
		},
		Regions: []builder.RegionConfig{
			{
				Region:           "ap-northeast-2",
				UsePublicSubnets: false,
				InstanceType:     "t3.medium",
				SshKey:           "test-master-key",
				AmiId:            "ami-01288945bd24ed49a",
				VPC:              "vpc-artd_apnortheast2",
				SecurityGroups: []string{
					fmt.Sprintf("%s-artd_apnortheast2", appName),
					"default-artd_apnortheast2",
				},
				HealthcheckTargetGroup: "hello-artdapne2-ext",
				TargetGroups: []string{
					"hello-artdapne2-ext",
				},
				AvailabilityZones: []string{
					"ap-northeast-2a",
					"ap-northeast-2b",
					"ap-northeast-2c",
				},
			},
		},
	})

	return yc
}
