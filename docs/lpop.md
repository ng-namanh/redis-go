# LPOP

```json metadata
{
  "title": "LPOP",
  "description": "Returns the first elements in a list after removing it. Deletes the list if the last element was popped.",
  "categories": ["docs","develop","stack","oss","rs","rc","oss","kubernetes","clients"],
  "arguments": [{"display_text":"key","key_spec_index":0,"name":"key","type":"key"},{"display_text":"count","name":"count","optional":true,"since":"6.2.0","type":"integer"}],
  "syntax_fmt": "LPOP key [count]",
  "complexity": "O(N) where N is the number of elements returned",
  "group": "list",
  "command_flags": ["write","fast"],
  "acl_categories": ["@write","@list","@fast"],
  "since": "1.0.0",
  "arity": -2,
  "key_specs": [{"RW":true,"access":true,"begin_search":{"spec":{"index":1},"type":"index"},"delete":true,"find_keys":{"spec":{"keystep":1,"lastkey":0,"limit":0},"type":"range"}}],
  "tableOfContents": {"sections":[{"id":"examples","title":"Examples"},{"id":"redis-software-and-redis-cloud-compatibility","title":"Redis Software and Redis Cloud compatibility"},{"id":"return-information","title":"Return information"}]}

,
  "codeExamples": [{"codetabsId":"cmds_list-steplpop","commands":[{"acl_categories":["@write","@list","@fast"],"complexity":"O(1)","name":"RPUSH"},{"acl_categories":["@write","@list","@fast"],"complexity":"O(N)","name":"LPOP"},{"acl_categories":["@read","@list","@slow"],"complexity":"O(S+N)","name":"LRANGE"}],"description":"Foundational: Remove and return the first element(s) from a list using LPOP (supports optional count parameter to pop multiple elements)","difficulty":"beginner","id":"lpop","languages":[{"id":"redis-cli","panelId":"panel_redis-cli_cmds_list-steplpop"},{"clientId":"redis-py","clientName":"redis-py","id":"Python","langId":"python","panelId":"panel_Python_cmds_list-steplpop"},{"id":"Node-js","panelId":"panel_Nodejs_cmds_list-steplpop"},{"clientId":"jedis","clientName":"Jedis","id":"Java-Sync","langId":"java","panelId":"panel_Java-Sync_cmds_list-steplpop"},{"clientId":"lettuce","clientName":"Lettuce","id":"Java-Async","langId":"java","panelId":"panel_Java-Async_cmds_list-steplpop"},{"clientId":"lettuce","clientName":"Lettuce","id":"Java-Reactive","langId":"java","panelId":"panel_Java-Reactive_cmds_list-steplpop"},{"clientId":"go-redis","clientName":"go-redis","id":"Go","langId":"go","panelId":"panel_Go_cmds_list-steplpop"},{"id":"dotnet-Sync (SE-Redis)","panelId":"panel_Csharp-Sync (SERedis)_cmds_list-steplpop"},{"clientId":"predis","clientName":"Predis","id":"PHP","langId":"php","panelId":"panel_PHP_cmds_list-steplpop"}]}]
}
```## Code Examples Legend

The code examples below show how to perform the same operations in different programming languages and client libraries:

- **Redis CLI**: Command-line interface for Redis
- **C# (Synchronous)**: StackExchange.Redis synchronous client
- **C# (Asynchronous)**: StackExchange.Redis asynchronous client
- **Go**: go-redis client
- **Java (Synchronous - Jedis)**: Jedis synchronous client
- **Java (Asynchronous - Lettuce)**: Lettuce asynchronous client
- **Java (Reactive - Lettuce)**: Lettuce reactive/streaming client
- **JavaScript (Node.js)**: node-redis client
- **PHP**: Predis client
- **Python**: redis-py client
- **Rust (Synchronous)**: redis-rs synchronous client
- **Rust (Asynchronous)**: redis-rs asynchronous client

Each code example demonstrates the same basic operation across different languages. The specific syntax and patterns vary based on the language and client library, but the underlying Redis commands and behavior remain consistent.

---

Removes and returns the first elements of the list stored at `key`.

By default, the command pops a single element from the beginning of the list.
When provided with the optional `count` argument, the reply will consist of up
to `count` elements, depending on the list's length.

## Examples

Foundational: Remove and return the first element(s) from a list using LPOP (supports optional count parameter to pop multiple elements)

**Difficulty:** Beginner

**Commands:** RPUSH, LPOP, LRANGE

**Complexity:**
- RPUSH: O(1)
- LPOP: O(N)
- LRANGE: O(S+N)

**Available in:** Redis CLI, C#, Go, Java (Asynchronous - Lettuce), Java (Reactive - Lettuce), Java (Synchronous - Jedis), JavaScript (Node.js), PHP, Python

##### Redis CLI

```
redis> RPUSH mylist "one" "two" "three" "four" "five"
(integer) 5
redis> LPOP mylist
"one"
redis> LPOP mylist 2
1) "two"
2) "three"
redis> LRANGE mylist 0 -1
1) "four"
2) "five"
```

##### C#

```csharp
        long lPopResult1 = db.ListRightPush("mylist", ["one", "two", "three", "four", "five"]);
        Console.WriteLine(lPopResult1); // >>> 5

        RedisValue lPopResult2 = db.ListLeftPop("mylist");
        Console.WriteLine(lPopResult2); // >>> one

        RedisValue[] lPopResult3 = db.ListLeftPop("mylist", 2);
        Console.WriteLine($"[{string.Join(", ", lPopResult3)}]");
        // >>> [two, three]

        RedisValue[] lPopResult4 = db.ListRange("mylist", 0, -1);
        Console.WriteLine($"[{string.Join(", ", lPopResult4)}]");
        // >>> [four, five]
```

##### Go

```go
	RPushResult, err := rdb.RPush(ctx,
		"mylist", "one", "two", "three", "four", "five",
	).Result()

	if err != nil {
		panic(err)
	}

	fmt.Println(RPushResult) // >>> 5

	lPopResult, err := rdb.LPop(ctx, "mylist").Result()

	if err != nil {
		panic(err)
	}

	fmt.Println(lPopResult) // >>> one

	lPopCountResult, err := rdb.LPopCount(ctx, "mylist", 2).Result()

	if err != nil {
		panic(err)
	}

	fmt.Println(lPopCountResult) // >>> [two three]

	lRangeResult, err := rdb.LRange(ctx, "mylist", 0, -1).Result()

	if err != nil {
		panic(err)
	}

	fmt.Println(lRangeResult) // >>> [four five]
```

##### Java (Asynchronous - Lettuce)

```java
            CompletableFuture<Void> lpop = asyncCommands.rpush("mylist", "one", "two", "three", "four", "five")
                    .thenCompose(res17 -> {
                        System.out.println(res17); // >>> 5

                        return asyncCommands.lpop("mylist");
                    }).thenCompose(res18 -> {
                        System.out.println(res18); // >>> one

                        return asyncCommands.lpop("mylist", 2);
                    }).thenCompose(res19 -> {
                        System.out.println(res19); // >>> [two, three]

                        return asyncCommands.lrange("mylist", 0, -1);
                    })
                    .thenAccept(res17_final -> System.out.println(res17_final)) // >>> [four, five]
                    .toCompletableFuture();
```

##### Java (Reactive - Lettuce)

```java
            Mono<Void> lpop = reactiveCommands.rpush("mylist", "one", "two", "three", "four", "five").doOnNext(res17 -> {
                System.out.println(res17); // >>> 5
            }).flatMap(res17 -> reactiveCommands.lpop("mylist")).doOnNext(res18 -> {
                System.out.println(res18); // >>> one
            }).flatMap(res18 -> reactiveCommands.lpop("mylist", 2).collectList()).doOnNext(res19 -> {
                System.out.println(res19); // >>> [two, three]
            }).flatMap(res19 -> reactiveCommands.lrange("mylist", 0, -1).collectList()).doOnNext(res17_final -> {
                System.out.println(res17_final); // >>> [four, five]
            }).then();
```

##### Java (Synchronous - Jedis)

```java
        long lPopResult1 = jedis.rpush(
            "mylist", "one", "two", "three", "four", "five"
        );
        System.out.println(lPopResult1); // >>> 5

        String lPopResult2 = jedis.lpop("mylist");
        System.out.println(lPopResult2); // >>> one

        List<String> lPopResult3 = jedis.lpop("mylist", 2);
        System.out.println(lPopResult3); // >>> [two, three]

        List<String> lPopResult4 = jedis.lrange("mylist", 0, -1);
        System.out.println(lPopResult4); // >>> [four, five]
```

##### JavaScript (Node.js)

```javascript
const res17 = await client.rPush('mylist', ["one", "two", "three", "four", "five"]);
console.log(res17); // 5

const res18 = await client.lPop('mylist');
console.log(res18); // 'one'

const res19 = await client.lPopCount('mylist', 2);
console.log(res19); // [ 'two', 'three' ]

const res20 = await client.lRange('mylist', 0, -1);
console.log(res20); // [ 'four', 'five' ]

```

##### PHP

```php
        $res17 = $r->rpush('mylist', 'one', 'two', 'three', 'four', 'five');
        echo $res17 . PHP_EOL;
        // >>> 5

        $res18 = $r->lpop('mylist');
        echo $res18 . PHP_EOL;
        // >>> one

        $res19 = $r->lpop('mylist', 2);
        echo json_encode($res19) . PHP_EOL;
        // >>> ["two","three"]

        $res20 = $r->lrange('mylist', 0, -1);
        echo json_encode($res20) . PHP_EOL;
        // >>> ["four","five"]
```

##### Python

```python
res17 = r.rpush("mylist", *["one", "two", "three", "four", "five"])
print(res17) # >>> 5

res18 = r.lpop("mylist")
print(res18) # >>> "one"

res19 = r.lpop("mylist", 2)
print(res19) # >>> ['two', 'three']

res17 = r.lrange("mylist", 0, -1)
print(res17)  # >>> [ "four", "five" ]

```



Give these commands a try in the interactive console:


RPUSH mylist "one" "two" "three" "four" "five"
LPOP mylist
LPOP mylist 2
LRANGE mylist 0 -1


## Redis Software and Redis Cloud compatibility

| Redis<br />Software | Redis<br />Cloud | <span style="min-width: 9em; display: table-cell">Notes</span> |
|:----------------------|:-----------------|:------|
| <span title="Supported">&#x2705; Standard</span><br /><span title="Supported"><nobr>&#x2705; Active-Active</nobr></span> | <span title="Supported">&#x2705; Standard</span><br /><span title="Supported"><nobr>&#x2705; Active-Active</nobr></span> |  |

## Return information

**RESP2:**

One of the following:
* [Nil reply](../../develop/reference/protocol-spec#bulk-strings): if the key does not exist.
* [Bulk string reply](../../develop/reference/protocol-spec#bulk-strings): when called without the _count_ argument, the value of the first element.
* [Array reply](../../develop/reference/protocol-spec#arrays): when called with the _count_ argument, a list of popped elements.

**RESP3:**

One of the following:
* [Null reply](../../develop/reference/protocol-spec#nulls): if the key does not exist.
* [Bulk string reply](../../develop/reference/protocol-spec#bulk-strings): when called without the _count_ argument, the value of the first element.
* [Array reply](../../develop/reference/protocol-spec#arrays): when called with the _count_ argument, a list of popped elements.



