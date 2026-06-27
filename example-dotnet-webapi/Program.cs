using System.Diagnostics;
using System.Text.Json;

var builder = WebApplication.CreateBuilder(args);

builder.Logging.ClearProviders();
builder.Logging.AddJsonConsole(options =>
{
    options.IncludeScopes = true;
    options.TimestampFormat = "yyyy-MM-ddTHH:mm:ss.fffzzz ";
    options.JsonWriterOptions = new JsonWriterOptions
    {
        Indented = false
    };
});

var app = builder.Build();

app.Use(async (context, next) =>
{
    var logger = context.RequestServices.GetRequiredService<ILoggerFactory>().CreateLogger("ExampleDotnetWebApi.Requests");
    var stopwatch = Stopwatch.StartNew();

    using var scope = logger.BeginScope(new Dictionary<string, object?>
    {
        ["TraceId"] = Activity.Current?.TraceId.ToString() ?? context.TraceIdentifier,
        ["RequestId"] = context.TraceIdentifier
    });

    try
    {
        await next();
        stopwatch.Stop();

        logger.LogInformation(
            "HTTP {Method} {Path} responded {StatusCode} in {ElapsedMilliseconds} ms",
            context.Request.Method,
            context.Request.Path.Value,
            context.Response.StatusCode,
            stopwatch.Elapsed.TotalMilliseconds);
    }
    catch (Exception exception)
    {
        stopwatch.Stop();

        logger.LogError(
            exception,
            "HTTP {Method} {Path} failed after {ElapsedMilliseconds} ms",
            context.Request.Method,
            context.Request.Path.Value,
            stopwatch.Elapsed.TotalMilliseconds);

        throw;
    }
});

var summaries = new[]
{
    "Freezing", "Bracing", "Chilly", "Cool", "Mild", "Warm", "Balmy", "Hot", "Sweltering", "Scorching"
};

app.MapGet("/", (ILogger<Program> logger) =>
{
    logger.LogInformation("Hello endpoint called at {ObservedAt}", DateTimeOffset.UtcNow);
    return Results.Ok(new
    {
        message = "Hello from the Sealog example Web API",
        weatherUrl = "/weatherforecast"
    });
});

app.MapGet("/weatherforecast", (ILogger<Program> logger) =>
{
    var forecast = Enumerable.Range(1, 5).Select(index =>
        new WeatherForecast
        (
            DateOnly.FromDateTime(DateTimeOffset.Now.AddDays(index).DateTime),
            Random.Shared.Next(-20, 55),
            summaries[Random.Shared.Next(summaries.Length)]
        ))
        .ToArray();

    logger.LogInformation(
        "Generated {ForecastCount} weather forecasts with temperatures from {MinimumTemperatureC}C to {MaximumTemperatureC}C",
        forecast.Length,
        forecast.Min(item => item.TemperatureC),
        forecast.Max(item => item.TemperatureC));

    return forecast;
})
.WithName("GetWeatherForecast");

app.Run();

record WeatherForecast(DateOnly Date, int TemperatureC, string? Summary)
{
    public int TemperatureF => 32 + (int)(TemperatureC / 0.5556);
}
