package windows_syslog_acceptance_test

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

func DeploymentName() string {
	return fmt.Sprintf("windows-syslog-tests-%d", GinkgoParallelNode())
}

func BoshCmd(args ...string) *gexec.Session {
	boshArgs := []string{"-n", "-d", DeploymentName()}
	boshArgs = append(boshArgs, args...)
	boshCmd := exec.Command("bosh", boshArgs...)
	By("Performing command: bosh " + strings.Join(boshArgs, " "))
	session, err := gexec.Start(boshCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())
	return session
}

func Cleanup() {
	BoshCmd("locks")
	session := BoshCmd("delete-deployment")
	Eventually(session, 10*time.Minute).Should(gexec.Exit(0))
	Eventually(BoshCmd("locks")).ShouldNot(gbytes.Say(DeploymentName()))
}

func Deploy(manifest string) *gexec.Session {
	session := BoshCmd("deploy", manifest, "-v", fmt.Sprintf("deployment=%s", DeploymentName()))
	Eventually(session, 10*time.Minute).Should(gexec.Exit(0))
	Eventually(BoshCmd("locks")).ShouldNot(gbytes.Say(DeploymentName()))
	return session
}

type LogOutput struct {
	Tables []struct {
		Rows []struct {
			Stdout string
		}
	}
}

func ForwardedLogs() string {
	return OutputFromBoshSSHCommand("storer", "cat /var/vcap/store/syslog_storer/syslog.log | grep windows || true")
}

func OutputFromBoshSSHCommand(job, command string) string {
	session := BoshCmd("ssh", job, "--json", "-r", "--opts=-T", "--command="+command)
	Eventually(session).Should(gexec.Exit(0))
	stdoutContents := session.Out.Contents()
	var logOutput LogOutput
	err := json.Unmarshal(stdoutContents, &logOutput)
	Expect(err).ToNot(HaveOccurred())
	return logOutput.Tables[0].Rows[0].Stdout
}

func WriteToTestFile(message string) func() string {
	return func() string {
		OutputFromBoshSSHCommand("forwarder", fmt.Sprintf("echo %s >> \"c:/var/vcap/sys/log/syslog_forwarder_windows/file.log\"", message))
		return ForwardedLogs()
	}
}
