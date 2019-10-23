#!/usr/bin/env bash
#GrafanaURL=http://128.199.96.206:8086/write?db=mydb
###### MULTI_MEMBERS
# Shard 0
if [ "$1" == "shard0-0" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnXfotCdLe7Gb8z13Qr2QqAneRYQPYN8faYTp8WLZCbd2JY2iSdB2qgxhhrkQ2PNZxrxZAj8944TjEEdjzPbhUTKUspxhA4vTFDW6aV" --nodemode "auto" --datadir "data/shard0-0" --listen "0.0.0.0:9434" --externaladdress "0.0.0.0:9434" --norpcauth --rpclisten "0.0.0.0:9334" --enablewallet --wallet "wallet1" --walletpassphrase "12345678" --walletautoinit --rpcwslisten "0.0.0.0:19334" 
fi
if [ "$1" == "shard0-1" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnXx5fajo5CJHWw9HVkHtXzZDzwffhPrwKEewEpoHb9prz1F1117DS6YWCXDkQo9zhmhkGMqHSgoJ2AozqhDeKTasi6vySMwDJ87Z55" --nodemode "auto" --datadir "data/shard0-1" --listen "0.0.0.0:9435" --externaladdress "0.0.0.0:9435" --norpcauth --rpclisten "0.0.0.0:9335" --rpcwslisten "0.0.0.0:19335" 
fi
if [ "$1" == "shard0-2" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnYJMhDwppXuLGDDQkqn3r7nne3G6nFuKWGAKTqtxEtQQSF4ChCNQKjPY1BCbx2FE7mX9LpdtDcJEcBfbDo3jnDLVsygPadA5NrqWCq" --nodemode "auto" --datadir "data/shard0-2" --listen "0.0.0.0:9436" --externaladdress "0.0.0.0:9436" --norpcauth --rpclisten "0.0.0.0:9336" 
fi
if [ "$1" == "shard0-3" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnYhQqyMBZLHXQDgZXQ5i51PuCyCqj9PKxy393rE8TfGFDLXufjr2fkJHtMxGhhfDZKrSzESaMHPHQLB8fmVbqAqpy3PBGCFCmUfH6b" --nodemode "auto" --datadir "data/shard0-3" --listen "0.0.0.0:9437" --externaladdress "0.0.0.0:9437" --norpcauth --rpclisten "0.0.0.0:9337" 
fi
# Shard 1
if [ "$1" == "shard1-0" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnXoEWG5H8x1odKxSj6sbLXowTBsVVkAxNWr5WnsbSTDkRiVrSdPy8QfMujntKRYBqywKMJCyhMpdr93T3XiUD5QJR1QFtTpYKpjBEx" --nodemode "auto" --datadir "data/shard1-0" --listen "0.0.0.0:9438" --externaladdress "0.0.0.0:9438" --norpcauth --rpclisten "0.0.0.0:9338" --enablewallet --wallet "wallet2" --walletpassphrase "12345678" --walletautoinit --rpcwslisten "127.0.0.1:19338" 
fi
if [ "$1" == "shard1-1" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnY7EyGkLjADf5zJtgWwUg1LMUrdQd6znLyhATKDeD7KrDq5zrBAvJQwiPgwYdJPfCNJdZydnzqBbVHYavNi13KY5Yhm1YxRSFQHdAF" --nodemode "auto" --datadir "data/shard1-1" --listen "0.0.0.0:9439" --externaladdress "0.0.0.0:9439" --norpcauth --rpclisten "0.0.0.0:9339" --rpcwslisten "127.0.0.1:19339" 
fi
if [ "$1" == "shard1-2" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnYRE1zDgXnormrqWFBDmnhey6WN3NBpviM3Qj1fZwvdFs7RxWYsaGXjpvBGbZ4HHARx7BW1ob1RqKZgLUsUq7NAoSjo7gRihpM6bfu" --nodemode "auto" --datadir "data/shard1-2" --listen "0.0.0.0:9440" --externaladdress "0.0.0.0:9440" --norpcauth --rpclisten "0.0.0.0:9340" 
fi
if [ "$1" == "shard1-3" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnYjhQ4KdEdgLVyuf8XYGFjrxqoqgSr9uVwAwRnR8g1A34woKjWsr26bsXZ6bJXKcwANxZTPcn5FVf9W7zbYp7tL4qeJVfSkaZXDji8" --nodemode "auto" --datadir "data/shard1-3" --listen "0.0.0.0:9441" --externaladdress "0.0.0.0:9441" --norpcauth --rpclisten "0.0.0.0:9341" 
fi
# Beacon
if [ "$1" == "beacon-0" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnX6USJnBzswUeuuanesuEEUGsxE8Pj3kkxkqvGRedUUPyocmtsqETX2WMBSvfBCwwsmMpxonhfQm2N5wy3SrNk11eYxEyDtwuGxw2E" --nodemode "auto" --datadir "data/beacon-0" --listen "0.0.0.0:9450" --externaladdress "0.0.0.0:9450" --norpcauth --rpclisten "0.0.0.0:9350" 
fi
if [ "$1" == "beacon-1" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnXWRThUTJQgoyH6evV8w19dFZfKWpCh8rZpfymW9JTgKPEVQS44nDRPpsooJiGStHxu81m3HA84t9DBVobz8hgBKRMcz2hddPWNX9N" --nodemode "auto" --datadir "data/beacon-1" --listen "0.0.0.0:9451" --externaladdress "0.0.0.0:9451" --norpcauth --rpclisten "0.0.0.0:9351" 
fi
if [ "$1" == "beacon-2" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnX2FzscSBNqNAMuQfeSQhMPamSAKDX9f7X7Nft6JR4hfxNh5WJ8r9jeAmbmzTytW8hvpeXVn9FwLD8fjuvn4k7oHtXeh2YxnMDUzZv" --nodemode "auto" --datadir "data/beacon-2" --listen "0.0.0.0:9452" --externaladdress "0.0.0.0:9452" --norpcauth --rpclisten "0.0.0.0:9352" 
fi
if [ "$1" == "beacon-3" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --privatekey "112t8rnXS3BtxiS9tW5e2pZB3Qvvvxp7eX8mt5dsqKwiWDgj2bxp49w2NbjhwTLXu7kuDbowTq7oe5guDpf2fQKHH2hs21Xra1Zx6jAaCCHZ" --nodemode "auto" --datadir "data/beacon-3" --listen "0.0.0.0:9453" --externaladdress "0.0.0.0:9453" --norpcauth --rpclisten "0.0.0.0:9353" 
fi
# FullNode
if [ "$1" == "full_node" ]; then
./incognito --discoverpeersaddress "0.0.0.0:9330" --nodemode "relay" --datadir "data/full_node" --listen "0.0.0.0:9454" --externaladdress "0.0.0.0:9454" --norpcauth --rpclisten "0.0.0.0:9354" --enablewallet --wallet "wallet_fullnode" --walletpassphrase "12345678" --walletautoinit --relayshards "all"  --txpoolmaxtx 100000
fi
######
if [ "$1" == "shard-candidate0-1" ]; then
./incognito --discoverpeersaddress "127.0.0.1:9330" --privatekey "112t8rnX2Fngsy1KuJv5GLLNUeB3gZwoHMss2cqRr1ECa2ibR7FQNUyE7kMFvq7rGtqVJULo8XfAxThuLxUwd8vv76MojbL3wPhxmTvbcd2S" --nodemode "auto" --datadir "data/shard-stake-1" --listen "127.0.0.1:9455" --externaladdress "127.0.0.1:9455" --norpcauth --rpclisten "127.0.0.1:9355"
fi
if [ "$1" == "shard-candidate0-2" ]; then
./incognito --discoverpeersaddress "127.0.0.1:9330" --privatekey "112t8rnXRrZ1gC7MYNFVUA1paZrE3iSiAb9AR7Z5quNBgR2ovrcfj8p4kTb3ynx6ddjnoPey3qA2vRiP17tCvpCHU9xBDwMq8D1Mg2GBM9eC" --nodemode "auto" --datadir "data/shard-stake-2" --listen "127.0.0.1:9456" --externaladdress "127.0.0.1:9456" --norpcauth --rpclisten "127.0.0.1:9356"
fi
if [ "$1" == "shard-candidate0-3" ]; then
./incognito --discoverpeersaddress "127.0.0.1:9330" --privatekey "112t8rnXbsJF4f5xtzM2jPW6dCYHChNksth7X64iUkLR4bGi2JBVjgJQRLeKRsdbiFaYMsxzrfbfKAp4TELGre45QkxHWCnwVXPGnnZjJKVL" --nodemode "auto" --datadir "data/shard-stake-6" --listen "0.0.0.0:9460" --externaladdress "0.0.0.0:9460" --norpcauth --rpclisten "0.0.0.0:9360"
fi
if [ "$1" == "shard-candidate1-1" ]; then
./incognito --discoverpeersaddress "127.0.0.1:9330" --privatekey "112t8rnY4DeSGZYb8r8sSN5WJr6ZL3NCafYAQ7f7Am9KXQDGSc3Qddpn7BfHW1i6CoVVk8vKEzJ25vA9uc9EdhoLU98eoUw7fMrPPrBdNB7Q" --nodemode "auto" --datadir "data/shard-stake-3" --listen "0.0.0.0:9457" --externaladdress "0.0.0.0:9457" --norpcauth --rpclisten "0.0.0.0:9357"
fi
if [ "$1" == "shard-candidate1-2" ]; then
./incognito --discoverpeersaddress "127.0.0.1:9330" --privatekey "112t8rnXBg35mhRzZDw7nNQCiRcM9QLg2DTHewY7kRZ3XCmgkdQa4iVcMUwMd5DTvvjEcCvv3SnCo5zSQpS93zskAxG6tdvR1QPBxtCmaBCK" --nodemode "auto" --datadir "data/shard-stake-4" --listen "0.0.0.0:9458" --externaladdress "0.0.0.0:9458" --norpcauth --rpclisten "0.0.0.0:9358"
fi
if [ "$1" == "shard-candidate1-3" ]; then
./incognito --discoverpeersaddress "127.0.0.1:9330" --privatekey "112t8rnXa3USo2wMdyuZTfurZrYe1ZJhqibnp9GHx8Jkf9dh5cU39sjqBTKoWPtNHvVZn2eqGj6V26PmELez85bUUBMBKG6tqFQrer2GkuJ4" --nodemode "auto" --datadir "data/shard-stake-5" --listen "0.0.0.0:9459" --externaladdress "0.0.0.0:9459" --norpcauth --rpclisten "0.0.0.0:9359"
fi
if [ "$1" == "shard-candidate1-4" ]; then
./incognito --discoverpeersaddress "127.0.0.1:9330" --privatekey "112t8rnXc38wPUsDdRasThfk58ME3KoSdeyXJS3tsyRuxPrSkvuc8MBT1gxiQjSCMBbifqibxEVAHimGcfnLbjVEEett8FtwFuBQ8zAUHTet" --nodemode "auto" --datadir "data/shard-stake-7" --listen "0.0.0.0:9461" --externaladdress "0.0.0.0:9461" --norpcauth --rpclisten "0.0.0.0:9361"
fi
if [ "$1" == "shard-candidate1-5" ]; then
./incognito --discoverpeersaddress "127.0.0.1:9330" --privatekey "112t8rnY6BgeU6bk7Nui1TwB2w2MsG3UeSXrgJ1n2tGVs66Qk9YT6E15PbXR4ai7eW1qyPrW7a2AUtN2otuXtBAZKtm2DDiUxh3ngZ6JPYHv" --nodemode "auto" --datadir "data/shard-stake-8" --listen "0.0.0.0:9462" --externaladdress "0.0.0.0:9462" --norpcauth --rpclisten "0.0.0.0:9362"
fi
