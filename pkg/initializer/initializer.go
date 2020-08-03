package initializer

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/DevopsArtFactory/goployer/pkg/tool"
	Logger "github.com/sirupsen/logrus"
)

var (
	manifestDir      = "manifests"
	scriptDir        = "scripts"
	scriptPath       = fmt.Sprintf("%s/%s", scriptDir, "userdata.sh")
	metricPath       = "metrics.yaml"
	manifestBase64   = "LS0tCm5hbWU6IGhlbGxvCnVzZXJkYXRhOgogIHR5cGU6IGxvY2FsCiAgcGF0aDogc2NyaXB0cy91c2VyZGF0YS5zaAoKYXV0b3NjYWxpbmc6ICZhdXRvc2NhbGluZ19wb2xpY3kKICAtIG5hbWU6IHNjYWxlX3VwCiAgICBhZGp1c3RtZW50X3R5cGU6IENoYW5nZUluQ2FwYWNpdHkKICAgIHNjYWxpbmdfYWRqdXN0bWVudDogMQogICAgY29vbGRvd246IDYwCiAgLSBuYW1lOiBzY2FsZV9kb3duCiAgICBhZGp1c3RtZW50X3R5cGU6IENoYW5nZUluQ2FwYWNpdHkKICAgIHNjYWxpbmdfYWRqdXN0bWVudDogLTEKICAgIGNvb2xkb3duOiAxODAKCmFsYXJtczogJmF1dG9zY2FsaW5nX2FsYXJtcwogIC0gbmFtZTogc2NhbGVfdXBfb25fdXRpbAogICAgbmFtZXNwYWNlOiBBV1MvRUMyCiAgICBtZXRyaWM6IENQVVV0aWxpemF0aW9uCiAgICBzdGF0aXN0aWM6IEF2ZXJhZ2UKICAgIGNvbXBhcmlzb246IEdyZWF0ZXJUaGFuT3JFcXVhbFRvVGhyZXNob2xkCiAgICB0aHJlc2hvbGQ6IDUwCiAgICBwZXJpb2Q6IDEyMAogICAgZXZhbHVhdGlvbl9wZXJpb2RzOiAyCiAgICBhbGFybV9hY3Rpb25zOgogICAgICAtIHNjYWxlX3VwCiAgLSBuYW1lOiBzY2FsZV9kb3duX29uX3V0aWwKICAgIG5hbWVzcGFjZTogQVdTL0VDMgogICAgbWV0cmljOiBDUFVVdGlsaXphdGlvbgogICAgc3RhdGlzdGljOiBBdmVyYWdlCiAgICBjb21wYXJpc29uOiBMZXNzVGhhbk9yRXF1YWxUb1RocmVzaG9sZAogICAgdGhyZXNob2xkOiAzMAogICAgcGVyaW9kOiAzMDAKICAgIGV2YWx1YXRpb25fcGVyaW9kczogMwogICAgYWxhcm1fYWN0aW9uczoKICAgICAgLSBzY2FsZV9kb3duCgp0YWdzOgogIC0gcHJvamVjdD10ZXN0CiAgLSBhcHA9aGVsbG8KICAtIHJlcG89aGVsbG8tZGVwbG95CgpzdGFja3M6CiAgLSBzdGFjazogYXJ0ZAogICAgcG9sbGluZ19pbnRlcnZhbDogMzBzCiAgICBhY2NvdW50OiBkZXYKICAgIGVudjogZGV2CiAgICBhc3N1bWVfcm9sZTogIiIKICAgIHJlcGxhY2VtZW50X3R5cGU6IEJsdWVHcmVlbgogICAgaWFtX2luc3RhbmNlX3Byb2ZpbGU6ICdhcHAtaGVsbG8tcHJvZmlsZScKICAgIGFuc2libGVfdGFnczogYWxsCiAgICBlYnNfb3B0aW1pemVkOiB0cnVlCiAgICBibG9ja19kZXZpY2VzOgogICAgICAtIGRldmljZV9uYW1lOiAvZGV2L3h2ZGEKICAgICAgICB2b2x1bWVfc2l6ZTogMTAwCiAgICAgICAgdm9sdW1lX3R5cGU6ICJncDIiCiAgICAgIC0gZGV2aWNlX25hbWU6IC9kZXYveHZkYgogICAgICAgIHZvbHVtZV90eXBlOiAic3QxIgogICAgICAgIHZvbHVtZV9zaXplOiA1MDAKICAgIGNhcGFjaXR5OgogICAgICBtaW46IDEKICAgICAgbWF4OiAyCiAgICAgIGRlc2lyZWQ6IDEKICAgIGF1dG9zY2FsaW5nOiAqYXV0b3NjYWxpbmdfcG9saWN5CiAgICBhbGFybXM6ICphdXRvc2NhbGluZ19hbGFybXMKICAgIGxpZmVjeWNsZV9jYWxsYmFja3M6CiAgICAgIHByZV90ZXJtaW5hdGVfcGFzdF9jbHVzdGVyOgogICAgICAgIC0gc2VydmljZSBoZWxsbyBzdG9wCgogICAgcmVnaW9uczoKICAgICAgLSByZWdpb246IGFwLW5vcnRoZWFzdC0yCiAgICAgICAgaW5zdGFuY2VfdHlwZTogdDMubWVkaXVtCiAgICAgICAgc3NoX2tleTogdGVzdC1tYXN0ZXIta2V5CiAgICAgICAgYW1pX2lkOiBhbWktMDEyODg5NDViZDI0ZWQ0OWEKICAgICAgICB1c2VfcHVibGljX3N1Ym5ldHM6IHRydWUKICAgICAgICB2cGM6IHZwYy1hcnRkX2Fwbm9ydGhlYXN0MgogICAgICAgIHNlY3VyaXR5X2dyb3VwczoKICAgICAgICAgIC0gaGVsbG8tYXJ0ZF9hcG5vcnRoZWFzdDIKICAgICAgICAgIC0gZGVmYXVsdC1hcnRkX2Fwbm9ydGhlYXN0MgogICAgICAgIGhlYWx0aGNoZWNrX3RhcmdldF9ncm91cDogaGVsbG8tYXJ0ZGFwbmUyLWV4dAogICAgICAgIGF2YWlsYWJpbGl0eV96b25lczoKICAgICAgICAgIC0gYXAtbm9ydGhlYXN0LTJhCiAgICAgICAgICAtIGFwLW5vcnRoZWFzdC0yYgogICAgICAgICAgLSBhcC1ub3J0aGVhc3QtMmMKICAgICAgICB0YXJnZXRfZ3JvdXBzOgogICAgICAgICAgLSBoZWxsby1hcnRkYXBuZTItZXh0Cg=="
	userdataBase64   = "IyEvYmluL2Jhc2gKCnN1ZG8geXVtIHVwZGF0ZQpzdWRvIGFtYXpvbi1saW51eC1leHRyYXMgaW5zdGFsbCBuZ2lueDEuMTIKc3VkbyBzZXJ2aWNlIG5naW54IHN0YXJ0"
	metricFileBase64 = "cmVnaW9uOiBhcC1ub3J0aGVhc3QtMgpzdG9yYWdlOgogIHR5cGU6IGR5bmFtb2RiCiAgbmFtZTogZ29wbG95ZXItbWV0cmljcwoK"
)

type Initializer struct {
	AppName string
	Logger  *Logger.Logger
}

func NewInitializer(appName string) Initializer {
	return Initializer{
		AppName: appName,
		Logger:  Logger.New(),
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

	i.CheckDir(filePath)

	i.Logger.Debugf("starts to write yaml configuration: %s", filePath)
	if err := generateFile(filePath, writeData); err != nil {
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
	tool.Blue.Fprintln(os.Stdout, "You have to put the right values on several parts with `<....>` in manifest file")
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
