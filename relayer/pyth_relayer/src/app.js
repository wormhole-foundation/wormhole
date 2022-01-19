const redis = require("redis");
(async () => {
  const client = redis.createClient();

  client.on("connect", function (err) {
    if (err) {
      console.error("Failed to connect to Redis:", err);
    } else {
      console.log("Redis Client Connected");
    }
  });

  console.log("Attempting to connect...");

  await client.connect();
  var cursor = "0";
  var iterations = 0;
  var finished = false;
  console.log("Entering scanRedis...");
  await client.set("dog", "fido");
  var retVal = await client.get("dog");
  if (retVal) {
    console.log("retVal:", retVal);
  } else {
    console.log("No retVal returned.");
  }
  var scanReply = await client.SCAN(cursor, "COUNT", "1");
  if (scanReply[0]) {
    console.log("new cursor value: ", scanReply);
  } else {
    console.log("No new cursor value");
  }
  for await (const si_key of client.scanIterator()) {
    const si_keyval = await client.get(si_key);
    console.log("SI: %s => %s", si_key, si_keyval);
  }
  console.log("Attempting to switch db...");
  await client.select(1);
  for await (const si_key of client.scanIterator()) {
    const si_keyval = await client.get(si_key);
    console.log("SI: %s => %s", si_key, si_keyval);
  }
  // var keysReply = await client.keys("*");
  // if (keysReply) {
  //   console.log("keysReply: ", keysReply);
  //   keysReply.forEach(async (element) => {
  //     var keyVal = await client.get(element);
  //     if (keyVal) {
  //       console.log("%s => %s", element, keyVal);
  //     } else {
  //       console.log("No keyVal returned.");
  //     }
  //   });
  // } else {
  //   console.log("No keysReply");
  // }
  // NOTE:  You cannot do select inside multi()
  // const [] = await redisClient
  // .multi()
  // .exec()
  await client.quit();
})();
