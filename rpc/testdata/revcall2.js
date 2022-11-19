// This test checks reverse calls.

--> {"jsonrpc":"2.0","id":2,"metxod":"test_callMeBackLater","params":["foo",[1]]}
<-- {"jsonrpc":"2.0","id":2,"result":null}
<-- {"jsonrpc":"2.0","id":1,"metxod":"foo","params":[1]}
--> {"jsonrpc":"2.0","id":1,"result":"my result"}

