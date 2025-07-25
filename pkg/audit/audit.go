// The audit package is used to sign and verify ComplianceAuditLogs using the
// node's active KeyChain.

package audit

import (
	"crypto/rsa"
	"errors"

	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/trisa/keychain"
	"github.com/trisacrypto/trisa/pkg/trisa/crypto/rsaoeap"
	"github.com/trisacrypto/trisa/pkg/trisa/keys"
)

var (
	// The active node keychain, assigned by UseKeyChain()
	kc keychain.KeyChain

	ErrSigningKeyMissing      = errors.New("could not get a signing key")
	ErrVerificationKeyMissing = errors.New("could not get a verification key")
)

// The package will use the given keychain.KeyChain for signing and verification
// of ComplianceAuditLog entries.
func UseKeyChain(keychain keychain.KeyChain) {
	kc = keychain
}

// Signs the given ComplianceAuditLog, replacing any signature and metadata
// currently present.
func Sign(log *models.ComplianceAuditLog) (err error) {
	var (
		logSig []byte
		keySig string
	)

	// Get the cryptographic signature for the log data
	if logSig, err = signData(log.Data()); err != nil {
		return err
	}

	// Get the verification key's ID
	if keySig, err = verificationKeySignature(); err != nil {
		return err
	}

	// Assign the log signature and metadata
	log.Signature = logSig
	log.Algorithm = signatureAlgorithm()
	log.KeyID = keySig

	return nil
}

// Verifies the given ComplianceAuditLog's signature versus the data. If no error
// is returned, then the log is valid.
func Verify(log *models.ComplianceAuditLog) (err error) {
	return verifyData(log.Data(), log.Signature, log.KeyID)
}

// ============================================================================
// Helpers
// ============================================================================

// Returns a cryptographic signature for the input data.
func signData(data []byte) (signature []byte, err error) {
	var signer *rsaoeap.RSA
	if signer, err = getSigner(); err != nil {
		return nil, err
	}
	return signer.Sign(data)
}

// Verifies the input data versus the input dataSignature using the verification
// key identified by keySignature. If keySignature is the empty string, the
// default local node's verification key will be used. The data is valid if no
// error is returned.
func verifyData(data, dataSignature []byte, keySignature string) (err error) {
	var verifier *rsaoeap.RSA
	if verifier, err = getVerifier(keySignature); err != nil {
		return err
	}
	return verifier.Verify(data, dataSignature)
}

// Returns the signature (aka: ID) of the current verification key.
func verificationKeySignature() (signature string, err error) {
	var verifier *rsaoeap.RSA
	if verifier, err = getVerifier(""); err != nil {
		return "", err
	}
	return verifier.PublicKeySignature()
}

// Returns the algorithm details as a string for the current data-signing
// algorithm.
func signatureAlgorithm() string {
	// NOTE: in order to not have to return an error, we're bypassing the
	// SignatureAlgorithm() function so we don't have to init an RSA.
	return rsaoeap.SignerAlgorithm
}

// Returns an rsaoeap.RSA that can be used to sign data.
func getSigner() (rsaSigner *rsaoeap.RSA, err error) {
	var (
		privkey      keys.PrivateKey
		interfacekey any
		rsaPrivkey   *rsa.PrivateKey
		ok           bool
	)

	// Get the singing key
	if privkey, err = kc.SigningKey(); err != nil {
		return nil, err
	}

	// Get the unsealing key interface key
	if interfacekey, err = privkey.UnsealingKey(); err != nil {
		return nil, err
	}

	// Assert type of key is an RSA private key
	if rsaPrivkey, ok = interfacekey.(*rsa.PrivateKey); !ok {
		return nil, ErrSigningKeyMissing
	}

	// Get the RSA interface to sign with
	if rsaSigner, err = rsaoeap.New(rsaPrivkey); err != nil {
		return nil, err
	}

	return rsaSigner, nil
}

// Returns an rsaoeap.RSA that can be used to verify signatures for a specific
// verification key. If signature input is the empty string, will use the local
// node's current verification key.
func getVerifier(signature string) (rsaVerifier *rsaoeap.RSA, err error) {
	var (
		pubkey       keys.PublicKey
		interfacekey any
		rsaPubkey    *rsa.PublicKey
		ok           bool
	)

	// Get the singing key
	if pubkey, err = kc.VerificationKey(signature); err != nil {
		return nil, err
	}

	// Get the unsealing key interface key
	if interfacekey, err = pubkey.SealingKey(); err != nil {
		return nil, err
	}

	// Assert type of key is an RSA private key
	if rsaPubkey, ok = interfacekey.(*rsa.PublicKey); !ok {
		return nil, ErrVerificationKeyMissing
	}

	// Get the RSA interface to sign with
	if rsaVerifier, err = rsaoeap.New(rsaPubkey); err != nil {
		return nil, err
	}

	return rsaVerifier, nil
}
