import { WebSocketClient } from "./index";

const client = new WebSocketClient({
    clientId: "test-client",
    url: "ws://localhost:8080/ws",
});

client.onConnect = () => {
    client.subscribe("data.created");
};

await client.connect();

await client.call("ping");

client.on("data.created", (event) => {
    console.log("data.created", event);
});

client.on("data.deleted", (event) => {
    console.log("data.deleted", event);
});
