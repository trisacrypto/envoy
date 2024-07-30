package postman_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/postman"
	"github.com/trisacrypto/trisa/pkg/ivms101"
)

func TestTransactionFromPayload(t *testing.T) {
	t.Run("Complete", func(t *testing.T) {
		payload, err := loadPayloadFixture("testdata/identity.pb.json", "testdata/transaction.pb.json")
		require.NoError(t, err, "could not load payload from fixtures")

		transaction := postman.TransactionFromPayload(payload)
		require.True(t, transaction.Originator.Valid)
		require.Equal(t, "Alessia Cremonesi", transaction.Originator.String)
		require.True(t, transaction.OriginatorAddress.Valid)
		require.Equal(t, "mrfAEzGzK23kU23FxrToDRPmV1ReNfX43G", transaction.OriginatorAddress.String)
		require.True(t, transaction.Beneficiary.Valid)
		require.Equal(t, "Alesia Sosa Calvillo", transaction.Beneficiary.String)
		require.True(t, transaction.BeneficiaryAddress.Valid)
		require.Equal(t, "n3Vgn8wF6ZkpKSe186NnytLPXdZ6j1JbHg", transaction.BeneficiaryAddress.String)
		require.Equal(t, "BTC", transaction.VirtualAsset)
		require.Equal(t, 0.46602501, transaction.Amount)
	})
}

func TestFindName(t *testing.T) {
	testCases := []struct {
		persons  []*ivms101.Person
		expected string
	}{
		{
			[]*ivms101.Person{},
			"",
		},
		{
			[]*ivms101.Person{makeLegalPerson(&ivms101.LegalPerson{})},
			"",
		},
		{
			[]*ivms101.Person{makeNaturalPerson(&ivms101.NaturalPerson{})},
			"",
		},
		{
			[]*ivms101.Person{
				makeLegalPersonWithNames(
					&ivms101.LegalPersonNameId{
						LegalPersonName:               "Acme Systems, Inc.",
						LegalPersonNameIdentifierType: ivms101.LegalPersonLegal,
					},
				),
			},
			"Acme Systems, Inc.",
		},
		{
			[]*ivms101.Person{
				makeNaturalPersonWithNames(
					&ivms101.NaturalPersonNameId{
						PrimaryIdentifier:   "Doe",
						SecondaryIdentifier: "John",
						NameIdentifierType:  ivms101.NaturalPersonLegal,
					},
				),
			},
			"John Doe",
		},
		{
			[]*ivms101.Person{
				makeLegalPersonWithNames(
					&ivms101.LegalPersonNameId{
						LegalPersonName:               "Acme Systems",
						LegalPersonNameIdentifierType: ivms101.LegalPersonShort,
					},
				),
			},
			"Acme Systems",
		},
		{
			[]*ivms101.Person{
				makeNaturalPersonWithNames(
					&ivms101.NaturalPersonNameId{
						PrimaryIdentifier:   "Doe",
						SecondaryIdentifier: "Rodrigo",
						NameIdentifierType:  ivms101.NaturalPersonBirth,
					},
				),
			},
			"Rodrigo Doe",
		},
		{
			[]*ivms101.Person{
				makeNaturalPersonWithNames(
					&ivms101.NaturalPersonNameId{
						PrimaryIdentifier:   "Jameson",
						SecondaryIdentifier: "",
						NameIdentifierType:  ivms101.NaturalPersonBirth,
					},
				),
			},
			"Jameson",
		},
		{
			[]*ivms101.Person{
				makeLegalPersonWithNames(
					&ivms101.LegalPersonNameId{
						LegalPersonName:               "Acme Systems",
						LegalPersonNameIdentifierType: ivms101.LegalPersonShort,
					},
					&ivms101.LegalPersonNameId{
						LegalPersonName:               "Acme Systems, Inc.",
						LegalPersonNameIdentifierType: ivms101.LegalPersonLegal,
					},
					&ivms101.LegalPersonNameId{
						LegalPersonName:               "Acme",
						LegalPersonNameIdentifierType: ivms101.LegalPersonTrading,
					},
				),
			},
			"Acme Systems, Inc.",
		},
		{
			[]*ivms101.Person{
				makeNaturalPersonWithNames(
					&ivms101.NaturalPersonNameId{
						PrimaryIdentifier:   "Doe",
						SecondaryIdentifier: "Rodrigo",
						NameIdentifierType:  ivms101.NaturalPersonBirth,
					},
					&ivms101.NaturalPersonNameId{
						PrimaryIdentifier:   "Gonzalez",
						SecondaryIdentifier: "Rodrigo",
						NameIdentifierType:  ivms101.NaturalPersonLegal,
					},
					&ivms101.NaturalPersonNameId{
						PrimaryIdentifier:   "Doe",
						SecondaryIdentifier: "John",
						NameIdentifierType:  ivms101.NaturalPersonMaiden,
					},
				),
			},
			"Rodrigo Gonzalez",
		},
	}

	for i, tc := range testCases {
		require.Equal(t, tc.expected, postman.FindName(tc.persons...), "test case %d failed", i)
	}
}

func TestFindAccount(t *testing.T) {
	testCases := []struct {
		person   any
		expected string
	}{
		{
			nil,
			"",
		},
		{
			&ivms101.Originator{},
			"",
		},
		{
			&ivms101.Beneficiary{},
			"",
		},
		{
			&ivms101.Originator{AccountNumbers: []string{"muw1FnY9EnU2FtbQ5AEVRaLpLCxQ5qo8k6"}},
			"muw1FnY9EnU2FtbQ5AEVRaLpLCxQ5qo8k6",
		},
		{
			&ivms101.Beneficiary{AccountNumbers: []string{"mi6ZedDjasreKsgMMJpC4asQUphDb4haHo"}},
			"mi6ZedDjasreKsgMMJpC4asQUphDb4haHo",
		},
		{
			&ivms101.Originator{AccountNumbers: []string{"", "", "muw1FnY9EnU2FtbQ5AEVRaLpLCxQ5qo8k6", ""}},
			"muw1FnY9EnU2FtbQ5AEVRaLpLCxQ5qo8k6",
		},
		{
			&ivms101.Beneficiary{AccountNumbers: []string{"", "mi6ZedDjasreKsgMMJpC4asQUphDb4haHo", "", ""}},
			"mi6ZedDjasreKsgMMJpC4asQUphDb4haHo",
		},
		{
			&ivms101.Originator{AccountNumbers: []string{"mo4pZniFJhiLzVddUHGcsr8ajXKtwyTa2p", "", "muw1FnY9EnU2FtbQ5AEVRaLpLCxQ5qo8k6", ""}},
			"mo4pZniFJhiLzVddUHGcsr8ajXKtwyTa2p",
		},
		{
			&ivms101.Beneficiary{AccountNumbers: []string{"", "mi6ZedDjasreKsgMMJpC4asQUphDb4haHo", "", "mfudqNyGL4JrihUayosXXzkLyDs1qYr5eQ"}},
			"mi6ZedDjasreKsgMMJpC4asQUphDb4haHo",
		},
		{
			int32(41),
			"",
		},
	}

	for i, tc := range testCases {
		require.Equal(t, tc.expected, postman.FindAccount(tc.person), "test case %d failed", i)
	}
}

//===========================================================================
// Helper Methods
//===========================================================================

func makeLegalPerson(person *ivms101.LegalPerson) *ivms101.Person {
	return &ivms101.Person{
		Person: &ivms101.Person_LegalPerson{
			LegalPerson: person,
		},
	}
}

func makeNaturalPerson(person *ivms101.NaturalPerson) *ivms101.Person {
	return &ivms101.Person{
		Person: &ivms101.Person_NaturalPerson{
			NaturalPerson: person,
		},
	}
}

func makeLegalPersonWithNames(names ...*ivms101.LegalPersonNameId) *ivms101.Person {
	return makeLegalPerson(&ivms101.LegalPerson{
		Name: &ivms101.LegalPersonName{
			NameIdentifiers: names,
		},
	})
}

func makeNaturalPersonWithNames(names ...*ivms101.NaturalPersonNameId) *ivms101.Person {
	return makeNaturalPerson(&ivms101.NaturalPerson{
		Name: &ivms101.NaturalPersonName{
			NameIdentifiers: names,
		},
	})
}
