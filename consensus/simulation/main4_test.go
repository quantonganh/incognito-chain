package main

import (
	"github.com/incognitochain/incognito-chain/consensus/blsbft"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"testing"
)

func Test_Main4Committee(t *testing.T) {

	committee := []string{
		"112t8rnXB47RhSdyVRU41TEf78nxbtWGtmjutwSp9YqsNaCpFxQGXcnwcXTtBkCGDk1KLBRBeWMvb2aXG5SeDUJRHtFV8jTB3weHEkbMJ1AL",
		"112t8rnXVdfBqBMigSs5fm9NSS8rgsVVURUxArpv6DxYmPZujKqomqUa2H9wh1zkkmDGtDn2woK4NuRDYnYRtVkUhK34TMfbUF4MShSkrCw5",
		"112t8rnXi8eKJ5RYJjyQYcFMThfbXHgaL6pq5AF5bWsDXwfsw8pqQUreDv6qgWyiABoDdphvqE7NFr9K92aomX7Gi5Nm1e4tEoV3qRLVdfSR",
		"112t8rnY42xRqJghQX3zvhgEa2ZJBwSzJ46SXyVQEam1yNpN4bfAqJwh1SsobjHAz8wwRvwnqJBfxrbwUuTxqgEbuEE8yMu6F14QmwtwyM43",
	}
	committeePkStruct := []incognitokey.CommitteePublicKey{}
	for _, v := range committee {
		p, _ := blsbft.LoadUserKeyFromIncPrivateKey(v)
		m, _ := blsbft.GetMiningKeyFromPrivateSeed(p)
		committeePkStruct = append(committeePkStruct, m.GetPublicKey())
	}
	n0 := NewNode(committeePkStruct, committee, 0)
	n1 := NewNode(committeePkStruct, committee, 1)
	n2 := NewNode(committeePkStruct, committee, 2)
	n3 := NewNode(committeePkStruct, committee, 3)

	n0.Start()
	n1.Start()
	n2.Start()
	n3.Start()

	//v := []int{	1, 1, 1, 1,/**/0, 0, 0, 0,/**/1, 1, 1, 1,/**/1, 1, 1, 1}
	//c := []int{	1, 1, 1, 1,/**/0, 0, 0, 0,/**/1, 1, 1, 1,/**/1, 1, 1, 1}
	select {}

}
