# 1 Reusable code for HHTP responses
 There is a branch for cleaing up the http response logic, check that and see if it's worth committing.

# 2 Give routing one authoritative implementation.
[server.go (line 118)](/Users/hjort/code/DistributedSystems/Lab1/server.go:118) and the pub/sub dispatcher spread routing across a map, method checks, and a switch. Meanwhile, isSupportedPath and isSupportedPubSubPath are exercised by tests but unused by the actual server.

A small route table containing the expected method and handler would consolidate those decisions. Test the real dispatch behavior. For files, replace the manual suffix loop with a map lookup using path.Ext. Its main benefit here is clarity; six extensi

# 3 Simplify broker methods and ordinary outcomes.
[broker_state.go (line 56)](/Users/hjort/code/DistributedSystems/Lab1/broker_state.go:56) has nested if/else blocks whose branches already return. Early returns would flatten these methods considerably.

Publishing to a topic without subscribers is normal behavior, so it need not manufacture an error that the caller immediately dismisses. Likewise, a normal polling timeout can be represented by a boolean rather than a freshly formatted error. Preserve sequence advancement even when nobody receives the publication.

# [main.go (line 20)](/Users/hjort/code/DistributedSystems/Lab1/main.go:20): usage and startup failures should be visible without enabling debug logging.
