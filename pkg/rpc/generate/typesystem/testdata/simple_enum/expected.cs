namespace MyApp.Models
{
    /// <summary>
    /// Available colors
    /// </summary>
    public static class Color
    {
        /// <summary>Red color</summary>
        public const string Red = "red";
        /// <summary>Green color</summary>
        public const string Green = "green";
        /// <summary>Blue color</summary>
        public const string Blue = "blue";

        /// <summary>
        /// Returns true if the value is a valid Color
        /// </summary>
        public static bool IsValid(string value)
        {
            switch (value)
            {
                case "red":
                case "green":
                case "blue":
                    return true;
                default:
                    return false;
            }
        }
    }
}
