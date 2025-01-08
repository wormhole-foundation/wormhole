// used to finish the DKG setup bysetting the private key into the secrets.json generatedby local DKG.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/certusone/wormhole/node/pkg/tss"
	"github.com/certusone/wormhole/node/pkg/tss/internal"
)

var secretkeypath = flag.String("key", "", "path to the secret key PEM file")
var lkgSecrets = flag.String("lkg", "", "path to the LKG secrets json file")

func main() {
	flag.Parse()

	if *lkgSecrets == "" || *secretkeypath == "" {
		flag.PrintDefaults()

		return
	}

	fmt.Println("loading lkg secrets from: " + *lkgSecrets)
	gsbts, err := os.ReadFile(*lkgSecrets)
	if err != nil {
		panic("couldn't load secrets from key generation protocol " + err.Error())
	}
	var gs tss.GuardianStorage
	json.Unmarshal(gsbts, &gs)

	fmt.Println("loading key from: " + *secretkeypath)
	bts, err := os.ReadFile(*secretkeypath)
	if err != nil {
		panic("issue reading secret key file" + err.Error())
	}

	// Cleaning up the pem file
	k, err := internal.PemToPrivateKey(bts)
	if err != nil {
		panic("issue parsing private key" + err.Error())
	}

	bts = internal.PrivateKeyToPem(k)

	gs.PrivateKey = bts

	fmt.Println("embbeding key into lkg secrets file")
	secrets, err := json.MarshalIndent(gs, "", "  ")
	if err != nil {
		panic("issue marshalling secrets" + err.Error())
	}

	if err := os.WriteFile(*lkgSecrets, secrets, 0644); err != nil {
		panic("couldn't write to file" + err.Error())
	}
}
