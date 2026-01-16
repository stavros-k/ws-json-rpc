import type { DataCreatedEvent, DataDeletedEvent } from "./events";
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
await client.subscribe("data.deleted");

const createdFn = (event: DataCreatedEvent) => {
    console.log("data.created", event);
};
const deletedFn = (event: DataDeletedEvent) => {
    console.log("data.deleted", event);
};
client.on("data.created", createdFn);
client.on("data.deleted", deletedFn);

client.off("data.created", createdFn);
client.off("data.deleted", deletedFn);

// No need to .off() the handlers, they are automatically removed when the event is unsubscribed
await client.unsubscribe("data.created");
await client.unsubscribe("data.deleted");
