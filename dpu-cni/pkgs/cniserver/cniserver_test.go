package cniserver_test

import (
	"net"
	"os"
	"path/filepath"

	"github.com/containernetworking/cni/pkg/skel"
	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	"github.com/openshift/dpu-operator/dpu-cni/pkgs/cni"
	"github.com/openshift/dpu-operator/dpu-cni/pkgs/cnihelper"
	"github.com/openshift/dpu-operator/dpu-cni/pkgs/cniserver"
	"github.com/openshift/dpu-operator/dpu-cni/pkgs/cnitypes"

	current "github.com/containernetworking/cni/pkg/types/100"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	utiltesting "k8s.io/client-go/util/testing"
	"k8s.io/klog/v2"
)

func processRequest(request *cnitypes.Request) (*current.Result, error) {
	// FIXME: Do actual work here.
	klog.Infof("DEBUG: %v", request)

	conf, err := cnihelper.ReadCNIConfig(request.Config)
	if err != nil {
		return nil, err
	}

	result := &current.Result{
		CNIVersion: conf.CNIVersion,
	}

	return result, nil
}

func PrepArgs(cniVersion string) *skel.CmdArgs {
	cniConfig := "{\"cniVersion\": \"" + cniVersion + "\",\"name\": \"dpucni\",\"type\": \"dpucni\"}"
	cmdArgs := &skel.CmdArgs{
		ContainerID: "fakecontainerid",
		Netns:       "fakenetns",
		IfName:      "fakeeth0",
		Args:        "",
		Path:        "fakepath",
		StdinData:   []byte(cniConfig),
	}
	os.Clearenv()
	os.Setenv("CNI_COMMAND", cnitypes.CNIAdd)
	os.Setenv("CNI_CONTAINERID", cmdArgs.ContainerID)
	os.Setenv("CNI_NETNS", cmdArgs.Netns)
	os.Setenv("CNI_IFNAME", cmdArgs.IfName)
	os.Setenv("CNI_PATH", cmdArgs.Path)

	return cmdArgs
}

var _ = g.Describe("Cniserver", func() {
	var (
		tmpDir           string
		plugin           *cni.Plugin
		cniServer        *cniserver.Server
		serverSocketPath string
		listener         net.Listener
	)

	g.Context("CNI Server APIs", func() {
		g.BeforeEach(func() {
			var err error
			// Create a tmp directory in the test container
			tmpDir, err = utiltesting.MkTmpdir("cniserver")
			o.Expect(err).NotTo(o.HaveOccurred())

			serverSocketPath = filepath.Join(tmpDir, filepath.Base(cnitypes.ServerSocketPath))
			cniServer = cniserver.NewCNIServer(
				cniserver.WithHandler(processRequest),
				cniserver.WithSocketPath(serverSocketPath))

			listener, err = cniServer.Listen()
			o.Expect(err).NotTo(o.HaveOccurred())

			go utilwait.Forever(func() {
				cniServer.Serve(listener)
			}, 0)

			plugin = &cni.Plugin{SocketPath: serverSocketPath}
		})

		g.AfterEach(func() {
			listener.Close()
			os.RemoveAll(tmpDir)
		})

		g.When("Normal ADD request", func() {
			cniVersion := "0.4.0"
			expectedResult := &current.Result{
				CNIVersion: cniVersion,
			}
			g.It("should get a correct response from the post request", func() {
				resp, ver, err := plugin.PostRequest(PrepArgs(cniVersion))
				o.Expect(err).NotTo(o.HaveOccurred())
				o.Expect(ver).To(o.Equal(cniVersion))
				o.Expect(resp.Result).To(o.Equal(expectedResult))
			})
		})
	})
})
