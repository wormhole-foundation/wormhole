package cmd

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"testing"
	"time"

	engine "github.com/certusone/wormhole/node/pkg/tss"
	"github.com/certusone/wormhole/node/pkg/tss/internal"
)

// create these from scrath, then store it into a single file.
// json should be able to read it.

var hostnames = []string{
	"t-gcp-threshsignnet-asia-01.gcp.testnet.xlabs.xyz",
	// "t-gcp-threshsignnet-asia-02.gcp.testnet.xlabs.xyz",
	// "t-gcp-threshsignnet-asia-03.gcp.testnet.xlabs.xyz",
	// "t-gcp-threshsignnet-asia-04.gcp.testnet.xlabs.xyz",

	"t-gcp-threshsignnet-usw-01.gcp.testnet.xlabs.xyz",
	// "t-gcp-threshsignnet-usw-02.gcp.testnet.xlabs.xyz",
	// "t-gcp-threshsignnet-usw-03.gcp.testnet.xlabs.xyz",
	// "t-gcp-threshsignnet-usw-04.gcp.testnet.xlabs.xyz",

	"t-gcp-threshsignnet-use-01.gcp.testnet.xlabs.xyz",
	// "t-gcp-threshsignnet-use-02.gcp.testnet.xlabs.xyz",
	// "t-gcp-threshsignnet-use-03.gcp.testnet.xlabs.xyz",
	// "t-gcp-threshsignnet-use-04.gcp.testnet.xlabs.xyz",

	"t-gcp-threshsignnet-euc-01.gcp.testnet.xlabs.xyz",
	// "t-gcp-threshsignnet-euc-02.gcp.testnet.xlabs.xyz",
	// "t-gcp-threshsignnet-euc-03.gcp.testnet.xlabs.xyz",
	// "t-gcp-threshsignnet-euc-04.gcp.testnet.xlabs.xyz",

	"t-gcp-threshsignnet-euw-01.gcp.testnet.xlabs.xyz",
	// "t-gcp-threshsignnet-euw-02.gcp.testnet.xlabs.xyz",
	// "t-gcp-threshsignnet-euw-03.gcp.testnet.xlabs.xyz",
}

const saveFile = "./lkg/lkg.json"
const specificKeysFolder = "5-servers"

type LKGConfig SetupConfigs

func TestMain(t *testing.T) {
	t.Skip("skipping main test, use specific tests instead")

	t.Run("CreateLKGConfigs", createLKGConfigs)

	t.Run("shoveKeysToPosition", shoveKeys)

	t.Run("storeGuardiansForTest", storeTestGuardians)

	t.Run("scpSecretsToServers", sendToServers)
}

func storeTestGuardians(t *testing.T) {
	cnfg := loadConfigs(t)

	mainFolder := "tss5"
	resultDir := path.Join("..", "..", "..", "internal", "testutils", "testdata", mainFolder)
	cleanResultFolder(t, resultDir)

	// if err := os.MkdirAll(_path, 0755); err != nil {
	// 	t.Fatalf("failed to create directory: %v", err)
	// }
	for i := range cnfg.Peers {
		// guardian := cnfg.Peers[i]
		saveLocation := cnfg.SaveLocation[i]
		if saveLocation == "" {
			t.Fatalf("guardian %d has empty WhereToSaveSecrets", i)
		}

		_path := path.Join("setkey", "keys", specificKeysFolder, saveLocation)
		lkgpath := path.Join(_path, "secrets.json")

		//read the file into a GuardianStorage struct
		gst, err := engine.NewGuardianStorageFromFile(lkgpath)
		if err != nil {
			t.Fatalf("failed to read guardian storage from file: %v", err)
		}

		fileIndex := gst.Self.CommunicationIndex
		fmt.Println("guardian index:", fileIndex)

		bts, err := json.MarshalIndent(gst, "", "  ")
		if err != nil {
			t.Fatalf("failed to marshal guardian storage: %v", err)
		}
		guardianFileName := fmt.Sprintf("guardian%d.json", fileIndex)
		guardianFilePath := path.Join(resultDir, guardianFileName)
		if err := os.WriteFile(guardianFilePath, bts, 0644); err != nil {
			t.Fatalf("failed to write guardian storage to file: %v", err)
		}
	}
}

func cleanResultFolder(t *testing.T, resultDir string) {
	// make sure the directory exists and EMPTY
	if err := os.MkdirAll(resultDir, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	// empty all files in the directory
	items, err := os.ReadDir(resultDir)
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}
	fmt.Println("directory contents before cleanup:")

	for _, listing := range items {
		name := listing.Name()

		if path.Ext(name) != ".json" || listing.IsDir() {
			fmt.Println("skipping file:", name)
			continue
		}

		fmt.Println("removing file:", listing.Name())
		filePath := path.Join(resultDir, listing.Name())
		if err := os.Remove(filePath); err != nil {
			t.Fatalf("failed to remove file %s: %v", filePath, err)
		}
	}
}

// scp -i ~/.ssh/id_ed25519 asia-01/secrets.json jonathan@%v:~
func sendToServers(t *testing.T) {
	cnfg := loadConfigs(t)

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not grab file due to runtime.Caller(0) failure")
	}

	errs := make(chan error, len(cnfg.Peers))

	workdir := path.Join(path.Dir(file), "setkey", "keys", specificKeysFolder)
	fmt.Println("sending files...")
	for i := range cnfg.Peers {
		go func(i int) {
			saveLocation := cnfg.SaveLocation[i]
			if saveLocation == "" {
				errs <- fmt.Errorf("guardian %d has empty WhereToSaveSecrets", i)

				return
			}

			localSecretsPath := path.Join(workdir, saveLocation, "secrets.json")
			hostname := cnfg.Peers[i].Hostname
			cmd := exec.Command("scp", "-i", "~/.ssh/id_ed25519", localSecretsPath, fmt.Sprintf("jonathan@%s:~", hostname))

			// fmt.Println(cmd.String())
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				errs <- fmt.Errorf("failed to run command: %v", err)

				return
			}
			fmt.Printf("sent %s to %s\n", localSecretsPath, hostname)

			errs <- nil
		}(i)
		// scp -i ~/.ssh/id_ed25519 <localpath> jonathan@t-gcp-threshsignnet-asia-01.gcp.testnet.xlabs.xyz:~
	}

	timer := time.NewTimer(time.Second * 90)
	defer timer.Stop()

	for i := 0; i < len(cnfg.Peers); i++ {
		select {
		case err := <-errs:
			if err != nil {
				t.Fatalf("failed to send file: %v", err)

				return
			}
		case <-timer.C:
			t.Fatalf("timed out waiting for file transfer")

			return
		}
	}

	fmt.Println("done sending files")
}

func loadConfigs(t *testing.T) LKGConfig {
	bts, err := os.ReadFile(saveFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	var cnfg LKGConfig
	if err := json.Unmarshal(bts, &cnfg); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}
	return cnfg
}

func shoveKeys(t *testing.T) {
	cnfg := loadConfigs(t)
	// TODO
	// var secretkeypath = flag.String("key", "", "path to the secret key PEM file")
	// var lkgSecrets = flag.String("lkg", "", "path to the LKG secrets json file")

	for i := range cnfg.Peers {

		saveLocation := cnfg.SaveLocation[i]
		if saveLocation == "" {
			t.Fatalf("guardian %d has empty WhereToSaveSecrets", i)
		}

		_path := path.Join(".", "setkey", "keys", specificKeysFolder, saveLocation)

		secretKey := cnfg.Secrets[i]
		keypath := path.Join(_path, "key.pem")
		if err := os.WriteFile(
			keypath,
			secretKey,
			0644,
		); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		lkgpath := path.Join(_path, "secrets.json")

		args := []string{
			"run", "./setkey",
			"--key=" + keypath,
			"--lkg=" + lkgpath,
		}

		cmd := exec.Command("go", args...)

		// Link the binary's stdout/stderr to your Go program's output
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to run command: %v", err)
		}
	}

	// setkey.Main([]string{
	// 	"-key", "../lkg/lkg.json",
	// 	"-lkg", saveFile,
	// })
}

func createLKGConfigs(t *testing.T) {
	if _, err := os.Stat(saveFile); err == nil {
		t.Fatalf("lkg.json already exists in lkg dir")
	} else if !os.IsNotExist(err) {
		t.Fatalf("unexpected error: %v", err)
	}

	cnfg := LKGConfig{
		NumParticipants: len(hostnames),
		WantedThreshold: 2*(len(hostnames)/3) + 1,
		Peers:           make([]Identifier, len(hostnames)),
		Secrets:         make([]engine.PEM, len(hostnames)),
		SaveLocation:    make([]string, len(hostnames)),
	}

	for i, hostname := range hostnames {
		sk, cert := createTLSCert(hostname)
		cnfg.Peers[i] = Identifier{
			Hostname: hostname,
			TlsX509:  cert,
		}
		cnfg.SaveLocation[i] = extractRegion(hostname)
		cnfg.Secrets[i] = internal.PrivateKeyToPem(sk)
	}

	bts, err := json.MarshalIndent(cnfg, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(saveFile, bts, 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
}

func extractRegion(domain string) string {
	parts := strings.Split(domain, "-")
	if len(parts) < 5 {
		panic("unexpected format")
	}

	regionPart := parts[3]               // "euw"
	rest := strings.Join(parts[4:], "-") // "01.gcp.testnet.xlabs.xyz"
	dotParts := strings.Split(rest, ".")
	if len(dotParts) == 0 {
		panic("unexpected format after region")
	}

	return fmt.Sprintf("%s-%s", regionPart, dotParts[0])
}

func createTLSCert(hostname string) (*ecdsa.PrivateKey, engine.PEM) {
	cert := createX509Cert(hostname)
	sk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}

	signedCert := internal.NewTLSCredentials(sk, cert)
	return sk, internal.CertToPem(signedCert)

}
func createX509Cert(hostname string) *x509.Certificate {
	// using random serial number
	var serialNumberLimit = new(big.Int).Lsh(big.NewInt(1), 128)

	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		panic(err)
	}

	tmpl := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{Organization: []string{"tsscomm"}},
		SignatureAlgorithm:    x509.ECDSAWithSHA256,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 366 * 40), // valid for > 40 years used for tests...
		BasicConstraintsValid: true,

		DNSNames:    []string{hostname, "localhost"},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1)},
	}
	return &tmpl
}
