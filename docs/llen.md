# LLEN

```json metadata
{
  "title": "LLEN",
  "description": "Returns the length of a list.",
  "categories": ["docs","develop","stack","oss","rs","rc","oss","kubernetes","clients"],
  "arguments": [{"display_text":"key","key_spec_index":0,"name":"key","type":"key"}],
  "syntax_fmt": "LLEN key",
  "complexity": "O(1)",
  "group": "list",
  "command_flags": ["readonly","fast"],
  "acl_categories": ["@read","@list","@fast"],
  "since": "1.0.0",
  "arity": 2,
  "key_specs": [{"RO":true,"begin_search":{"spec":{"index":1},"type":"index"},"find_keys":{"spec":{"keystep":1,"lastkey":0,"limit":0},"type":"range"}}],
  "tableOfContents": {"sections":[{"id":"examples","title":"Examples"},{"id":"redis-software-and-redis-cloud-compatibility","title":"Redis Software and Redis Cloud compatibility"},{"id":"return-information","title":"Return information"}]}

,
  "codeExamples": [{"codetabsId":"cmds_list-stepllen","commands":[{"acl_categories":["@write","@list","@fast"],"complexity":"O(1)","name":"LPUSH"},{"acl_categories":["@read","@list","@fast"],"complexity":"O(1)","name":"LLEN"}],"description":"Foundational: Get the length of a list using LLEN (returns 0 if key doesn\u0026amp;#39;t exist, useful for checking list size)","difficulty":"beginner","id":"llen","languages":[{"id":"redis-cli","panelId":"panel_redis-cli_cmds_list-stepllen"},{"clientId":"redis-py","clientName":"redis-py","id":"Python","langId":"python","panelId":"panel_Python_cmds_list-stepllen"},{"id":"Node-js","panelId":"panel_Nodejs_cmds_list-stepllen"},{"clientId":"jedis","clientName":"Jedis","id":"Java-Sync","langId":"java","panelId":"panel_Java-Sync_cmds_list-stepllen"},{"clientId":"lettuce","clientName":"Lettuce","id":"Java-Async","langId":"java","panelId":"panel_Java-Async_cmds_list-stepllen"},{"clientId":"lettuce","clientName":"Lettuce","id":"Java-Reactive","langId":"java","panelId":"panel_Java-Reactive_cmds_list-stepllen"},{"clientId":"go-redis","clientName":"go-redis","id":"Go","langId":"go","panelId":"panel_Go_cmds_list-stepllen"},{"id":"dotnet-Sync (SE-Redis)","panelId":"panel_Csharp-Sync (SERedis)_cmds_list-stepllen"},{"clientId":"predis","clientName":"Predis","id":"PHP","langId":"php","panelId":"panel_PHP_cmds_list-stepllen"}]}]
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

Returns the length of the list stored at `key`.
If `key` does not exist, it is interpreted as an empty list and `0` is returned.
An error is returned when the value stored at `key` is not a list.

## Examples

Foundational: Get the length of a list using LLEN (returns 0 if key doesn't exist, useful for checking list size)

**Difficulty:** Beginner

**Commands:** LPUSH, LLEN

**Complexity:**
- LPUSH: O(1)
- LLEN: O(1)

**Available in:** Redis CLI, C#, Go, Java (Asynchronous - Lettuce), Java (Reactive - Lettuce), Java (Synchronous - Jedis), JavaScript (Node.js), PHP, Python

##### Redis CLI

```
redis> LPUSH mylist "World"
(integer) 1
redis> LPUSH mylist "Hello"
(integer) 2
redis> LLEN mylist
(integer) 2
```

##### C#

```csharp
        long lLenResult1 = db.ListLeftPush("mylist", "World");
        Console.WriteLine(lLenResult1); // >>> 1

        long lLenResult2 = db.ListLeftPush("mylist", "Hello");
        Console.WriteLine(lLenResult2); // >>> 2

        long lLenResult3 = db.ListLength("mylist");
        Console.WriteLine(lLenResult3); // >>> 2
```

##### Go

```go
	lPushResult1, err := rdb.LPush(ctx, "mylist", "World").Result()

	if err != nil {
		panic(err)
	}

	fmt.Println(lPushResult1) // >>> 1

	lPushResult2, err := rdb.LPush(ctx, "mylist", "Hello").Result()

	if err != nil {
		panic(err)
	}

	fmt.Println(lPushResult2) // >>> 2

	lLenResult, err := rdb.LLen(ctx, "mylist").Result()

	if err != nil {
		panic(err)
	}

	fmt.Println(lLenResult) // >>> 2
```

##### Java (Asynchronous - Lettuce)

```java
            CompletableFuture<Void> llen = asyncCommands.lpush("mylist", "World").thenCompose(res11 -> {
                System.out.println(res11); // >>> 1

                return asyncCommands.lpush("mylist", "Hello");
            }).thenCompose(res12 -> {
                System.out.println(res12); // >>> 2

                return asyncCommands.llen("mylist");
            })
                    .thenAccept(res13 -> System.out.println(res13)) // >>> 2
                    .toCompletableFuture();
```

##### Java (Reactive - Lettuce)

```java
            Mono<Void> llen = reactiveCommands.lpush("mylist", "World").doOnNext(res11 -> {
                System.out.println(res11); // >>> 1
            }).flatMap(res11 -> reactiveCommands.lpush("mylist", "Hello")).doOnNext(res12 -> {
                System.out.println(res12); // >>> 2
            }).flatMap(res12 -> reactiveCommands.llen("mylist")).doOnNext(res13 -> {
                System.out.println(res13); // >>> 2
            }).then();
```

##### Java (Synchronous - Jedis)

```java
        long lLenResult1 = jedis.lpush("mylist", "World");
        System.out.println(lLenResult1); // >>> 1

        long lLenResult2 = jedis.lpush("mylist", "Hello");
        System.out.println(lLenResult2); // >>> 2

        long lLenResult3 = jedis.llen("mylist");
        System.out.println(lLenResult3); // >>> 2
```

##### JavaScript (Node.js)

```javascript
const res11 = await client.lPush('mylist', 'World');
console.log(res11); // 1

const res12 = await client.lPush('mylist', 'Hello');
console.log(res12); // 2

const res13 = await client.lLen('mylist');
console.log(res13); // 2

```

##### PHP

```php
        $res11 = $r->lpush('mylist', 'World');
        echo $res11 . PHP_EOL;
        // >>> 1

        $res12 = $r->lpush('mylist', 'Hello');
        echo $res12 . PHP_EOL;
        // >>> 2

        $res13 = $r->llen('mylist');
        echo $res13 . PHP_EOL;
        // >>> 2
```

##### Python

```python
res11 = r.lpush("mylist", "World")
print(res11) # >>> 1

res12 = r.lpush("mylist", "Hello")
print(res12) # >>> 2

res13 = r.llen("mylist")
print(res13)  # >>> 2

```



Give these commands a try in the interactive console:


LPUSH mylist "World"
LPUSH mylist "Hello"
LLEN mylist


## Redis Software and Redis Cloud compatibility

| Redis<br />Software | Redis<br />Cloud | <span style="min-width: 9em; display: table-cell">Notes</span> |
|:----------------------|:-----------------|:------|
| <span title="Supported">&#x2705; Standard</span><br /><span title="Supported"><nobr>&#x2705; Active-Active</nobr></span> | <span title="Supported">&#x2705; Standard</span><br /><span title="Supported"><nobr>&#x2705; Active-Active</nobr></span> |  |

## Return information

**RESP2:**

[Integer reply](../../develop/reference/protocol-spec#integers): the length of the list.

**RESP3:**

[Integer reply](../../develop/reference/protocol-spec#integers): the length of the list.



