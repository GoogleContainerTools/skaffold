package core

func newChallenge(challengeType AcmeChallenge, token string) Challenge {
	return Challenge{
		Type:   challengeType,
		Status: StatusPending,
		Token:  token,
	}
}

// HTTPChallenge01 constructs a http-01 challenge.
func HTTPChallenge01(token string) Challenge {
	return newChallenge(ChallengeTypeHTTP01, token)
}

// DNSChallenge01 constructs a dns-01 challenge.
func DNSChallenge01(token string) Challenge {
	return newChallenge(ChallengeTypeDNS01, token)
}

// TLSALPNChallenge01 constructs a tls-alpn-01 challenge.
func TLSALPNChallenge01(token string) Challenge {
	return newChallenge(ChallengeTypeTLSALPN01, token)
}

// DNSAccountChallenge01 constructs a dns-account-01 challenge.
func DNSAccountChallenge01(token string) Challenge {
	return newChallenge(ChallengeTypeDNSAccount01, token)
}
