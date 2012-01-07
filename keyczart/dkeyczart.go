package main

import (
	"../_obj/dkeyczar"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
)

func Save(location string, km dkeyczar.KeyManager, crypter dkeyczar.Crypter) {

	err := os.Mkdir(location, 0700)

	if err != nil {
		fmt.Println("unable to create key directory: " + err.Error())
		return
	}

	Update(location, km, crypter)
}

func Update(location string, km dkeyczar.KeyManager, crypter dkeyczar.Crypter) {

	s := km.ToJSONs(crypter)

	ioutil.WriteFile(location+"/meta", []byte(s[0]), 0600)

	for i := 1; i < len(s); i++ {
		fname := location + "/" + strconv.Itoa(i)
		ioutil.WriteFile(fname, []byte(s[i]), 0600)
	}
}

func main() {

	optLocation := flag.String("location", "", "the location of the keys")
	optSize := flag.Int("size", 0, "the size of key to generate")
	optName := flag.String("name", "", "the name of the key")
	optPurpose := flag.String("purpose", "", "the purpose of the key (crypt/sign)")
	optCrypter := flag.String("crypter", "", "the key to use when dealing with encrypted keys")
	optAsymmetric := flag.String("asymmetric", "", "the asymmetric algorithm to use (dsa/rsa)")
	optStatus := flag.String("status", "", "the status (active/primary/inactive)")
	optVersion := flag.Int("version", 0, "the version of the key to use")
	optDestination := flag.String("destination", "", "the location to store the exported keys")

	flag.Parse()

	command := flag.Arg(0)

	if command == "" {
		flag.Usage()
		return
	}

	var crypter dkeyczar.Crypter

	if *optCrypter != "" {
		fmt.Println("using crypter: ", *optCrypter)
		r := dkeyczar.NewFileReader(*optCrypter)
		crypter, _ = dkeyczar.NewCrypter(r)
	}

	km := dkeyczar.NewKeyManager()

	if command != "create" {

		if *optLocation == "" {
			fmt.Println("missing required --location argument")
			return
		}

		lr := dkeyczar.NewFileReader(*optLocation)

		if crypter != nil {
			fmt.Println("decrypting keys..")
			lr = dkeyczar.NewEncryptedReader(lr, crypter)
		}

		err := km.Load(lr)
		if err != nil {
			fmt.Println("failed to load key: ", err)
			return
		}
	}

	if command == "create" {
		// make sure location doesn't exist

		keypurpose := dkeyczar.P_TEST

		switch *optPurpose {
		case "crypt":
			keypurpose = dkeyczar.P_DECRYPT_AND_ENCRYPT
		case "sign":
			keypurpose = dkeyczar.P_SIGN_AND_VERIFY
		default:
			fmt.Println("unknown cryptographic purpose: ", *optPurpose)
			return
		}

		if *optAsymmetric != "" && *optAsymmetric != "dsa" && *optAsymmetric != "rsa" {
			fmt.Println("unknown asymmetric key type: ", *optAsymmetric)
			return
		}

		keytype := dkeyczar.T_AES

		switch {
		case keypurpose == dkeyczar.P_DECRYPT_AND_ENCRYPT && *optAsymmetric == "":
			keytype = dkeyczar.T_AES
		case keypurpose == dkeyczar.P_DECRYPT_AND_ENCRYPT && *optAsymmetric == "rsa":
			keytype = dkeyczar.T_RSA_PRIV
		case keypurpose == dkeyczar.P_SIGN_AND_VERIFY && *optAsymmetric == "":
			keytype = dkeyczar.T_HMAC_SHA1
		case keypurpose == dkeyczar.P_SIGN_AND_VERIFY && *optAsymmetric == "rsa":
			keytype = dkeyczar.T_RSA_PRIV
		case keypurpose == dkeyczar.P_SIGN_AND_VERIFY && *optAsymmetric == "dsa":
			keytype = dkeyczar.T_DSA_PRIV
		default:
			fmt.Println("unknown or invalid purpose/asymmetric combination: ", *optPurpose, "/", *optAsymmetric)
			return
		}

		km.Create(*optName, keypurpose, keytype)

		Save(*optLocation, km, crypter)

	} else if command == "promote" {
		km.Promote(*optVersion)
		Update(*optLocation, km, crypter)
	} else if command == "demote" {
		km.Demote(*optVersion)
		Update(*optLocation, km, crypter)
	} else if command == "addkey" {
		status := dkeyczar.S_ACTIVE
		if *optStatus == "" {
			// FIXME: really, want to do: status = (km.kz.primary == -1 ? S_PRIMARY : S_ACTIVE)
			status = dkeyczar.S_ACTIVE
		} else if *optStatus == "primary" {
			status = dkeyczar.S_PRIMARY
		} else if *optStatus == "active" {
			status = dkeyczar.S_ACTIVE
		} else if *optStatus == "inactive" {
			status = dkeyczar.S_INACTIVE
		} else {
			fmt.Println("unknown status: ", *optStatus)
		}

		err := km.AddKey(uint(*optSize), status)
		if err != nil {
			fmt.Println("error adding key: ", err)
			return
		}
		Update(*optLocation, km, crypter)
	} else if command == "pubkey" {
		kpub := km.PubKeys()
		Save(*optDestination, kpub, nil) // doesn't make sense to encrypt a public key
	}

}
