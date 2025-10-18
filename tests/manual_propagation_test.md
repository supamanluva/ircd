# Phase 7.4 Manual Propagation Test

This document describes how to manually verify that all Phase 7.4 message routing and propagation features work correctly.

## Setup

1. Start the hub server:
```bash
go run cmd/ircd/main.go -config configs/hub.yaml &
```

2. Start leaf server 1:
```bash
go run cmd/ircd/main.go -config configs/leaf1.yaml &
```

3. Start leaf server 2:
```bash
go run cmd/ircd/main.go -config configs/leaf2.yaml &
```

4. Wait 5 seconds for auto-connect to establish links

## Test 1: PRIVMSG Routing

**Terminal 1** (connected to Hub on port 6667):
```
NICK Alice
USER alice 0 * :Alice User
```

**Terminal 2** (connected to Leaf1 on port 6668):
```
NICK Bob
USER bob 0 * :Bob User
```

**Terminal 1** (Alice sends message to Bob):
```
PRIVMSG Bob :Hello from hub!
```

**Expected**: Terminal 2 (Bob) should receive:
```
:Alice!alice@<host> PRIVMSG Bob :Hello from hub!
```

## Test 2: JOIN Propagation

**Terminal 1** (Alice on Hub):
```
JOIN #test
```

**Terminal 2** (Bob on Leaf1):
```
JOIN #test
```

**Expected**: Both users should see each other's JOIN messages.

## Test 3: Channel Message Routing

**Terminal 1** (Alice in #test):
```
PRIVMSG #test :Hello channel!
```

**Expected**: Terminal 2 (Bob in #test) should receive:
```
:Alice!alice@<host> PRIVMSG #test :Hello channel!
```

## Test 4: TOPIC Propagation

**Terminal 1** (Alice):
```
TOPIC #test :New topic set by Alice
```

**Expected**: Terminal 2 (Bob) should receive:
```
:Alice!alice@<host> TOPIC #test :New topic set by Alice
```

## Test 5: MODE Propagation

**Terminal 1** (Alice gives Bob ops):
```
MODE #test +o Bob
```

**Expected**: Terminal 2 (Bob) should receive:
```
:Alice!alice@<host> MODE #test +o Bob
```

## Test 6: KICK Propagation

**Terminal 3** (connected to Leaf2 on port 6669):
```
NICK Charlie
USER charlie 0 * :Charlie User
JOIN #test
```

**Terminal 1** (Alice kicks Charlie):
```
KICK #test Charlie :Kicked by Alice
```

**Expected**: Terminal 3 (Charlie) should receive:
```
:Alice!alice@<host> KICK #test Charlie :Kicked by Alice
```

## Test 7: INVITE Propagation

**Terminal 3** (Charlie):
```
JOIN #private
MODE #private +i
```

**Terminal 1** (Alice):
```
INVITE Bob #private
```

**Expected**: Terminal 2 (Bob) should receive:
```
:Alice!alice@<host> INVITE Bob :#private
```

## Test 8: NICK Propagation

**Terminal 2** (Bob changes nick):
```
NICK Robert
```

**Expected**: Terminal 1 (Alice in #test) should receive:
```
:Bob!bob@<host> NICK Robert
```

## Test 9: QUIT Propagation

**Terminal 3** (Charlie):
```
QUIT :Goodbye!
```

**Expected**: All users in channels with Charlie should receive:
```
:Charlie!charlie@<host> QUIT :Goodbye!
```

## Test 10: SQUIT (Operator Only)

**Terminal 1** (Alice becomes operator):
```
OPER admin <password>
SQUIT leaf1.example.com :Testing disconnect
```

**Expected**:
- Leaf1 server should disconnect
- All users from Leaf1 should QUIT with netsplit message
- Hub log should show server disconnection

## Verification

All tests should demonstrate that messages are properly:
1. Routed from source server to destination server
2. Delivered to the correct users/channels
3. Include proper source information (nick!user@host)
4. Maintain network state consistency

## Automated Test Status

The automated integration test (`tests/test_phase7.4_integration.sh`) verifies:
- ✅ Test 1: Server linking (3 servers)
- ✅ Test 9: Error handling (stability)
- ✅ Test 10: Network state consistency

Tests 2-8 require manual verification or improved test methodology using proper IRC clients.
