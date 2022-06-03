package main

import (
	"encoding/hex"
	"flag"

	"github.com/certusone/wormhole/node/pkg/notify/discord"
	"github.com/certusone/wormhole/node/pkg/vaa"
	"go.uber.org/zap"
)

var (
	botToken = flag.String("botToken", "", "Discord bot token")
)

func init() {
	flag.Parse()

	if *botToken == "" {
		panic("please provide bot token")
	}
}

const (
	exampleVaaBytes = `01000000010d01b074d1f0e483942e2e222121749b94d82696cb2692f455b6efa5ee5ffe7644382ec92c69a01e2815d07e86ea79cba64d0db0797cd7fea7184f1b6386470f15c40002e327ba5b53500f73f33dc5d499e3483eb97b69e5c7c338a57f01eff7884f74443e10b0f5e895fd92392448662ceb788e00bcbd4af54129ca1386a34c94e4a91e00033076b5dbcf0826cf245848cb0d66aa556bd63de37a02dbc282b8b8559057071b675844eff803a201ac40d4b4c203f51c56b6a7879831507d052ab5df5a62c5f40004973bd450a72d74960b3adb7345fc2bf66e57ebf60e31599999ee45c2ee31d8656812047289e4ff72dcbad211acf96008b019dad22d26d90c923509769cb1c12601061a7f9cb619addccdda4f79493945506ea6622ddf07be15b9012a5eb694b330a465b23a7eb6ff20715d5b36f73af372ab27a6015cd37b60b833c8574ea84dcfdb000732cd1559a554908d77b6e6ee539de392236ab2f2274554ff4e59761927cc2ce71b41a7f72dd5b91fe41a04361e71c4589b659c48652d7fea135d926ef50fe6e90109de5789414b8dd2eacd3eb1bbf29842aa1c55fc1f8449e0da61cb63ea161c0d9c52a796e79b365cf9bda8fac18a322de54c3e4f32039f26a222b0a7aa374e9d08010a0e43548171d384415d9d1c931c3950e2cfd4416b944cd144ca283b243e765e8a35bc3f3c8aab91f121dd15bc0a337fb0b5938f273aacbb1693f7f010d9e6ea88000c1ccd493f9512f3c1a8042a0b568f389c6e457c61a52aafd5b8f3915d63ab270745e1b6adfae19a005699dcdb4885e95d5bc72d8de8f7219d47d6af3882dfdc9c000d5359bf248f08afb1fd3ecce0b014c4eae7fc51f0dffb5f38536cce2b11becce80150c44d051281f350d4d47666c7b161d9da341a938872aeaa0f4cf21d52229a000e2b206ab8f5bcc8833716631626ecfd5b2b287c47c967025b22e03706eb5dba8f7e7c8c59f12167650d7ce871938c9053ddcb826db823951f88e811dfdc43d1fe001065bf71105ce76db70c75542d5dec9b45df756a021190165ee8a41b1ed9410665318e7b9fc9d68411084e40f67c4fe717f4949a480e6006ea09e29710492356aa0012ac1c80da05f99eaf5edae8daaf40b1161bd6733d3b4ca9764f7ca980c31e475a450d151404708ed465b2d25577d1f9ce1662c735b14d0e9978068964786570f100615c4f9700005e650001ec7372995d5cc8732397fb0ad35c0121e0eaa90d26f828a534cab54391b3a4f500000000000005d4200100000000000000000000000000000000000000000000000000000005883f8260000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb4800020000000000000000000000009200cd14071a98cda2ab3a87f94973aa44cbbf1600020000000000000000000000000000000000000000000000000000000000000000`
)

func main() {
	b, err := hex.DecodeString(exampleVaaBytes)
	if err != nil {
		panic(err)
	}

	v, err := vaa.Unmarshal(b)
	if err != nil {
		panic(err)
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	d, err := discord.NewDiscordNotifier(
		*botToken, "alerts", logger)
	if err != nil {
		logger.Fatal("failed to initialize notifier", zap.Error(err))
	}

	if err := d.MissingSignaturesOnObservation(v, 14, 13, true, []string{
		"Certus One", "Not Certus One"}); err != nil {
		logger.Fatal("failed to send test message", zap.Error(err))
	}

	if err := d.MissingSignaturesOnObservation(v, 14, 13, true, []string{
		"Certus One"}); err != nil {
		logger.Fatal("failed to send test message", zap.Error(err))
	}
}
