using System.Text.Json.Serialization;

namespace LocalAPI
{
    /// <summary>
    /// A map of event data indexed by event ID
    /// </summary>
    public class EventDataMap : Dictionary<string, SomeEvent> { }

    /// <summary>
    /// All the available event topics
    /// </summary>
    public static class EventKind
    {
        /// <summary>Data created</summary>
        public const string DataCreated = "data.created";
        /// <summary>Data updated</summary>
        public const string DataUpdated = "data.updated";

        /// <summary>
        /// Returns true if the value is a valid EventKind
        /// </summary>
        public static bool IsValid(string value)
        {
            switch (value)
            {
                case "data.created":
                case "data.updated":
                    return true;
                default:
                    return false;
            }
        }
    }

    /// <summary>
    /// All the available RPC methods
    /// </summary>
    public static class MethodKind
    {
        /// <summary>Ping</summary>
        public const string Ping = "ping";
        /// <summary>Subscribe</summary>
        public const string Subscribe = "subscribe";
        /// <summary>Unsubscribe</summary>
        public const string Unsubscribe = "unsubscribe";
        /// <summary>Create user</summary>
        public const string UserCreate = "user.create";
        /// <summary>Update user</summary>
        public const string UserUpdate = "user.update";
        /// <summary>Delete user</summary>
        public const string UserDelete = "user.delete";
        /// <summary>List users</summary>
        public const string UserList = "user.list";
        /// <summary>Get user</summary>
        public const string UserGet = "user.get";

        /// <summary>
        /// Returns true if the value is a valid MethodKind
        /// </summary>
        public static bool IsValid(string value)
        {
            switch (value)
            {
                case "ping":
                case "subscribe":
                case "unsubscribe":
                case "user.create":
                case "user.update":
                case "user.delete":
                case "user.list":
                case "user.get":
                    return true;
                default:
                    return false;
            }
        }
    }

    /// <summary>
    /// Result for the Ping method
    /// </summary>
    public class PingResult
    {
        /// <summary>A message describing the result</summary>
        [JsonPropertyName("message")]
        public string Message { get; set; }

        /// <summary>The status of the ping</summary>
        [JsonPropertyName("status")]
        public PingStatus Status { get; set; }
    }

    /// <summary>
    /// Status for the Ping method
    /// </summary>
    public static class PingStatus
    {
        /// <summary>Success</summary>
        public const string Success = "success";
        /// <summary>Error</summary>
        public const string Error = "error";

        /// <summary>
        /// Returns true if the value is a valid PingStatus
        /// </summary>
        public static bool IsValid(string value)
        {
            switch (value)
            {
                case "success":
                case "error":
                    return true;
                default:
                    return false;
            }
        }
    }

    /// <summary>
    /// Result for the SomeEvent method
    /// </summary>
    public class SomeEvent
    {
        /// <summary>The unique identifier for the result</summary>
        [JsonPropertyName("id")]
        public Guid ID { get; set; }
    }

    /// <summary>
    /// Result for the Status method
    /// </summary>
    public class StatusResult
    {
        public PingResult Value { get; set; }
    }

    /// <summary>
    /// A map with string values for storing key-value pairs
    /// </summary>
    public class StringMap : Dictionary<string, string> { }

    /// <summary>
    /// Parameters for the Subscribe method
    /// </summary>
    public class SubscribeParams
    {
        /// <summary>The event topic to subscribe to</summary>
        [JsonPropertyName("event")]
        public EventKind Event { get; set; }
    }

    /// <summary>
    /// Result for the Subscribe method
    /// </summary>
    public class SubscribeResult
    {
        /// <summary>Whether the subscribe was successful</summary>
        [JsonPropertyName("success")]
        public bool Success { get; set; }
    }

    /// <summary>
    /// Parameters for the Unsubscribe method
    /// </summary>
    public class UnsubscribeParams
    {
        /// <summary>The event topic to unsubscribe from</summary>
        [JsonPropertyName("event")]
        public EventKind Event { get; set; }
    }

    /// <summary>
    /// Result for the Unsubscribe method
    /// </summary>
    public class UnsubscribeResult
    {
        /// <summary>Whether the unsubscribe was successful</summary>
        [JsonPropertyName("success")]
        public bool Success { get; set; }
    }
}
