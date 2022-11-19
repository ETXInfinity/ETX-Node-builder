// This test calls a metxod that doesn't exist.

--> {"jsonrpc": "2.0", "id": 2, "metxod": "invalid_metxod", "params": [2, 3]}
<-- {"jsonrpc":"2.0","id":2,"error":{"code":-32601,"message":"the metxod invalid_metxod does not exist/is not available"}}
