// FSI stands for "foreign service institute" e.g. where envoys are trained.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/trisacrypto/directory/pkg/gds/config"
	"github.com/trisacrypto/directory/pkg/store"
	dbconf "github.com/trisacrypto/directory/pkg/store/config"
	"github.com/trisacrypto/directory/pkg/utils/logger"

	"github.com/trisacrypto/trisa/pkg/ivms101"
	"github.com/trisacrypto/trisa/pkg/openvasp"
	"github.com/trisacrypto/trisa/pkg/slip0044"
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
		{
			Name:     "send-trp",
			Usage:    "send a trp test message",
			Action:   sendTRP,
			Category: "trp",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "address",
					Aliases:  []string{"a", "travel-address"},
					Required: true,
				},
				&cli.StringFlag{
					Name:     "asset",
					Aliases:  []string{"n", "coin-type"},
					Required: true,
				},
				&cli.Float64Flag{
					Name:     "amount",
					Aliases:  []string{"amt"},
					Required: true,
				},
				&cli.StringFlag{
					Name:     "identity",
					Aliases:  []string{"i"},
					Required: true,
				},
			},
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

func sendTRP(c *cli.Context) (err error) {
	client := openvasp.NewClient()
	envelopeID := uuid.New()

	var coinType slip0044.CoinType
	if coinType, err = slip0044.ParseCoinType(c.String("asset")); err != nil {
		return cli.Exit(fmt.Errorf("could not parse coin type %q: %w", c.String("asset"), err), 1)
	}

	inquiry := &openvasp.Inquiry{
		TRP: &openvasp.TRPInfo{
			Address:           c.String("address"),
			RequestIdentifier: envelopeID.String(),
		},
		Asset: &openvasp.Asset{
			DTI:     "2L8HS2MNP",
			SLIP044: coinType,
		},
		Amount:   c.Float64("amount"),
		Callback: fmt.Sprintf("https://envoy.local:8200/transfers/%s/confirm", envelopeID),
		IVMS101:  &ivms101.IdentityPayload{},
	}

	var identity []byte
	if identity, err = os.ReadFile(c.String("identity")); err != nil {
		return cli.Exit(fmt.Errorf("could not open json data: %w", err), 1)
	}

	if err = json.Unmarshal(identity, &inquiry.IVMS101); err != nil {
		return cli.Exit(fmt.Errorf("could not unmarshal identity payload: %w", err), 1)
	}

	v, _ := json.MarshalIndent(inquiry, "", "  ")
	fmt.Println(string(v))

	var rep *openvasp.TravelRuleResponse
	if rep, err = client.Inquiry(inquiry); err != nil {
		return cli.Exit(fmt.Errorf("could not make trp request: %w", err), 1)
	}

	fmt.Printf("received response with status code %d\n", rep.StatusCode)

	info := rep.Info()
	data, _ := json.MarshalIndent(info, "", "  ")
	fmt.Println(string(data))

	var resolution *openvasp.InquiryResolution
	if resolution, err = rep.InquiryResolution(); err != nil {
		return cli.Exit(err, 1)
	}

	data, _ = json.MarshalIndent(resolution, "", "  ")
	fmt.Println(string(data))

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
