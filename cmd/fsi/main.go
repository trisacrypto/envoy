// FSI stands for "foreign service institute" e.g. where envoys are trained.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/trisacrypto/directory/pkg/gds/config"
	"github.com/trisacrypto/directory/pkg/store"
	dbconf "github.com/trisacrypto/directory/pkg/store/config"
	"github.com/trisacrypto/directory/pkg/utils/logger"

	"github.com/trisacrypto/trisa/pkg/ivms101"
	pb "github.com/trisacrypto/trisa/pkg/trisa/gds/models/v1beta1"

	"github.com/trisacrypto/envoy/pkg"

	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
)

var (
	db   store.Store
	conf config.Config
)

func main() {
	godotenv.Load()

	app := cli.NewApp()
	app.Name = "fsi"
	app.Usage = "initialize the local development environment for testing purposes"
	app.Version = pkg.Version()
	app.Flags = []cli.Flag{}
	app.Commands = []*cli.Command{
		{
			Name:     "init-gds",
			Usage:    "populate the local GDS with the docker-compose node information",
			Action:   initGDS,
			Before:   connectDB,
			After:    closeDB,
			Category: "localhost",
		},
		{
			Name:     "inspect",
			Usage:    "check the contents of the local GDS",
			Action:   inspectGDS,
			Before:   connectDB,
			After:    closeDB,
			Category: "localhost",
		},
	}

	app.Run(os.Args)
}

//===========================================================================
// Localhost Actions
//===========================================================================

func initGDS(c *cli.Context) (err error) {
	ctx := context.Background()
	envoyID, counterpartyID := "", ""

	// Create VASP record for envoy node
	if envoyID, err = db.CreateVASP(ctx, envoyVASP()); err != nil {
		return cli.Exit(fmt.Errorf("could not create envoy record: %w", err), 1)
	}
	fmt.Printf("created envoy record in local gds with id: %s\n", envoyID)

	// Create VASP record for counterparty node
	if counterpartyID, err = db.CreateVASP(ctx, counterpartyVASP()); err != nil {
		return cli.Exit(fmt.Errorf("could not create counterparty record: %w", err), 1)
	}
	fmt.Printf("created counterparty record in local gds with id: %s\n", counterpartyID)

	return nil
}

func envoyVASP() *pb.VASP {
	return &pb.VASP{
		RegisteredDirectory: "trisatest.dev",
		Entity: &ivms101.LegalPerson{
			Name: &ivms101.LegalPersonName{
				NameIdentifiers: []*ivms101.LegalPersonNameId{
					{
						LegalPersonName:               "Localhost Development",
						LegalPersonNameIdentifierType: ivms101.LegalPersonLegal,
					},
				},
				LocalNameIdentifiers:    nil,
				PhoneticNameIdentifiers: nil,
			},
			GeographicAddresses: []*ivms101.Address{
				{
					AddressType: ivms101.AddressTypeBusiness,
					AddressLine: []string{
						"1803 Welsh Bush Rd",
						"Utica, MN 55104",
					},
					Country: "US",
				},
			},
			CustomerNumber: "376128278645689",
			NationalIdentification: &ivms101.NationalIdentification{
				NationalIdentifier:     "0FOH00SEASDBQDSGOI84",
				NationalIdentifierType: ivms101.NationalIdentifierLEIX,
				CountryOfIssue:         "",
				RegistrationAuthority:  "",
			},
			CountryOfRegistration: "US",
		},
		Contacts:            &pb.Contacts{},
		IdentityCertificate: nil,
		SigningCertificates: nil,
		CommonName:          "envoy.local",
		TrisaEndpoint:       "envoy.local:8100",
		Website:             "http://envoy.local:8000",
		BusinessCategory:    pb.BusinessCategoryBusiness,
		VaspCategories:      []string{"DEX", "Exchange"},
		EstablishedOn:       "2024-06-05",
		Trixo:               &pb.TRIXOQuestionnaire{},
		VerificationStatus:  pb.VerificationState_VERIFIED,
		VerifiedOn:          time.Now().Format(time.RFC3339),
		FirstListed:         time.Now().Format(time.RFC3339),
		LastUpdated:         time.Now().Format(time.RFC3339),
		Signature:           nil,
		Version:             nil,
		Extra:               nil,
		CertificateWebhook:  "",
		NoEmailDelivery:     true,
	}
}

func counterpartyVASP() *pb.VASP {
	return &pb.VASP{
		RegisteredDirectory: "trisatest.dev",
		Entity: &ivms101.LegalPerson{
			Name: &ivms101.LegalPersonName{
				NameIdentifiers: []*ivms101.LegalPersonNameId{
					{
						LegalPersonName:               "Localhost Counterparty",
						LegalPersonNameIdentifierType: ivms101.LegalPersonLegal,
					},
				},
				LocalNameIdentifiers:    nil,
				PhoneticNameIdentifiers: nil,
			},
			GeographicAddresses: []*ivms101.Address{
				{
					AddressType: ivms101.AddressTypeBusiness,
					AddressLine: []string{
						"Markische Strasse 75",
						"Dortmund 44141",
						"North Rhine-Westphalia",
					},
					Country: "DE",
				},
			},
			CustomerNumber: "2149535420055041",
			NationalIdentification: &ivms101.NationalIdentification{
				NationalIdentifier:     "2T3800PLME5FJEPUKZ74",
				NationalIdentifierType: ivms101.NationalIdentifierLEIX,
				CountryOfIssue:         "",
				RegistrationAuthority:  "",
			},
			CountryOfRegistration: "DE",
		},
		Contacts:            &pb.Contacts{},
		IdentityCertificate: nil,
		SigningCertificates: nil,
		CommonName:          "counterparty.local",
		TrisaEndpoint:       "counterparty.local:9100",
		Website:             "http://counterparty.local:9000",
		BusinessCategory:    pb.BusinessCategoryBusiness,
		VaspCategories:      []string{"DEX", "Exchange"},
		EstablishedOn:       "2024-06-05",
		Trixo:               &pb.TRIXOQuestionnaire{},
		VerificationStatus:  pb.VerificationState_VERIFIED,
		VerifiedOn:          time.Now().Format(time.RFC3339),
		FirstListed:         time.Now().Format(time.RFC3339),
		LastUpdated:         time.Now().Format(time.RFC3339),
		Signature:           nil,
		Version:             nil,
		Extra:               nil,
		CertificateWebhook:  "",
		NoEmailDelivery:     true,
	}
}

func inspectGDS(c *cli.Context) (err error) {
	iter := db.ListVASPs(context.Background())
	for iter.Next() {
		vasp, err := iter.VASP()
		if err != nil {
			return cli.Exit(err, 1)
		}
		fmt.Printf("%s %s\n", vasp.CommonName, vasp.VerificationStatus)
	}

	return nil
}

//===========================================================================
// Before and After
//===========================================================================

func configure(*cli.Context) (err error) {
	conf = config.Config{
		DirectoryID: "trisatest.dev",
		Maintenance: true,
		Database: dbconf.StoreConfig{
			URL:           "leveldb:///tmp/gds/db",
			ReindexOnBoot: false,
			Insecure:      true,
		},
	}

	logger.Discard()
	return nil
}

func connectDB(c *cli.Context) (err error) {
	// Configure the connection to the local database
	if err = configure(c); err != nil {
		return err
	}

	// Connect to the trtl server and create a store to access data directly like GDS
	if db, err = store.Open(conf.Database); err != nil {
		return cli.Exit(fmt.Errorf("could not open store: %w", err), 1)
	}
	return nil
}

func closeDB(c *cli.Context) (err error) {
	if err = db.Close(); err != nil {
		return cli.Exit(err, 2)
	}
	return nil
}
