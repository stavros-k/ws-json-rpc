using System.Text.Json.Serialization;

namespace MyApp.Models
{
    /// <summary>
    /// A person entity
    /// </summary>
    public class Person
    {
        /// <summary>Person's age</summary>
        [JsonPropertyName("age")]
        public long Age { get; set; }

        /// <summary>Person's email address</summary>
        [JsonPropertyName("email")]
        public string Email { get; set; }

        /// <summary>Person's name</summary>
        [JsonPropertyName("name")]
        public string Name { get; set; }
    }
}
