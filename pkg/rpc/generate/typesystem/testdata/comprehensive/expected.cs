using System.Text.Json.Serialization;

namespace MyApp.Models
{
    /// <summary>
    /// User status
    /// </summary>
    public static class Status
    {
        /// <summary>User is active</summary>
        public const string Active = "active";
        /// <summary>User is inactive</summary>
        public const string Inactive = "inactive";

        /// <summary>
        /// Returns true if the value is a valid Status
        /// </summary>
        public static bool IsValid(string value)
        {
            switch (value)
            {
                case "active":
                case "inactive":
                    return true;
                default:
                    return false;
            }
        }
    }

    /// <summary>
    /// A string map
    /// </summary>
    public class StringMap : Dictionary<string, string> { }

    /// <summary>
    /// A list of tags
    /// </summary>
    public class Tags : List<string> { }

    /// <summary>
    /// User entity
    /// </summary>
    public class User
    {
        /// <summary>When the user was created</summary>
        [JsonPropertyName("createdAt")]
        public DateTime CreatedAt { get; set; }

        /// <summary>User ID</summary>
        [JsonPropertyName("id")]
        public UserID ID { get; set; }

        /// <summary>User status</summary>
        [JsonPropertyName("status")]
        public Status Status { get; set; }

        /// <summary>User tags</summary>
        [JsonPropertyName("tags")]
        public List<string> Tags { get; set; }
    }

    /// <summary>
    /// Unique identifier for a user
    /// </summary>
    public class UserID
    {
        public Guid Value { get; set; }
    }
}
