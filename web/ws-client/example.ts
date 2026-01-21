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

await client.connect();

await client.call("ping");
await client.subscribe("data.created");

const createdFn = (event: DataCreated) => {
    console.log("data.created", event);
};

client.on("data.created", createdFn);
client.off("data.created", createdFn);

// No need to .off() the handlers, they are automatically removed when the event is unsubscribed
await client.unsubscribe("data.created");
