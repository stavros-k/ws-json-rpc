import type { DataCreated } from "./generated";
import { WebSocketClient } from "./index";

const client = new WebSocketClient({
    clientId: "test-client",
    url: "ws://localhost:8080/ws",
});

client.onReconnectAttempt = (attempt) => console.log(`Reconnect attempt ${attempt}`);
client.onDisconnect = () => console.log("Disconnected");
client.onError = (error) => console.error("Error", error);
client.onConnect = () => console.log("Connected");

// Connect to the WebSocket server
await client.connect();

// Call a method on the server without parameters
const res = await client.call("ping");
// Type narrowing, if there is error, result
if (res.error) {
    // If there is an error, .result is undefined
    console.error("Failed to ping server", res.error);
} else {
    // If there is no error, .result is defined
    console.log("Ping response", res.result);
}

// Automatically subscribes to the event on first listener addition
const detach1 = await client.addEventListener("data.created", (event) => {
    // Event is typed here
    console.log("data.created", event.id);
});
const detach2 = await client.addEventListener("data.created", () => {
    console.log("Another handler for data.created");
});

detach1();
// Automatically unsubscribes from the event on last listener removal
detach2();

// You can also manually subscribe, and unsubscribe
await client.subscribe("data.created");

// Pass a bound function as listener and manually remove it
const myFunc = (event: DataCreated) => console.log(event.id);
await client.addEventListener("data.created", myFunc);
// Can't pass inline function here, need to pass the same reference
client.removeEventListener("data.created", myFunc);

await client.unsubscribe("data.created");
