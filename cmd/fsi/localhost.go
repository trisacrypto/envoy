package main

import (
	pb "github.com/trisacrypto/trisa/pkg/trisa/gds/models/v1beta1"
)

func envoyVASP() (vasp *pb.VASP, err error) {
	vasp = new(pb.VASP)
	if err = unmarshalPBFixture("localhost/envoy.pb.json", vasp); err != nil {
		return nil, err
	}
	return vasp, nil
}

func counterpartyVASP() (vasp *pb.VASP, err error) {
	vasp = new(pb.VASP)
	if err = unmarshalPBFixture("localhost/counterparty.pb.json", vasp); err != nil {
		return nil, err
	}
	return vasp, nil
}
