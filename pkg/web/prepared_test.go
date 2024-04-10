package web_test

import (
	"fmt"

	"github.com/trisacrypto/trisa/pkg/openvasp/traddr"
)

func ExampleBeneficiaryTravelAddress() {
	ta, err := traddr.Encode("//api.bob.vaspbot.net:443/?t=i")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(ta)
	// Output:
	//ta24sKtCmQ92AFBJjkbFW6DxDigXsqZQbyq7k1ePJpN57gffo
}
