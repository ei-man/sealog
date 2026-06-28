using System.Diagnostics;
using System.Text.Json;
using Microsoft.ApplicationInsights;
using Microsoft.ApplicationInsights.DataContracts;
using Microsoft.ApplicationInsights.Extensibility;
using Microsoft.Extensions.Logging.ApplicationInsights;

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
builder.Logging.AddApplicationInsights();
builder.Logging.AddFilter<ApplicationInsightsLoggerProvider>(string.Empty, LogLevel.Information);

var appInsightsConnectionString = builder.Configuration["APPLICATIONINSIGHTS_CONNECTION_STRING"]
    ?? builder.Configuration["ApplicationInsights:ConnectionString"];

builder.Services.AddApplicationInsightsTelemetry(options =>
{
    options.ConnectionString = appInsightsConnectionString;
});
builder.Services.AddSingleton(_ =>
{
    var configuration = TelemetryConfiguration.CreateDefault();
    configuration.ConnectionString = appInsightsConnectionString;
    return new TelemetryClient(configuration);
});

var app = builder.Build();

app.Use(async (context, next) =>
{
    var logger = context.RequestServices.GetRequiredService<ILoggerFactory>().CreateLogger("ExampleDotnetWebApi.Requests");
    var telemetry = context.RequestServices.GetRequiredService<TelemetryClient>();
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

        telemetry.TrackTrace(
            $"HTTP {context.Request.Method} {context.Request.Path.Value} responded {context.Response.StatusCode}",
            SeverityLevel.Information,
            new Dictionary<string, string>
            {
                ["Method"] = context.Request.Method,
                ["Path"] = context.Request.Path.Value ?? string.Empty,
                ["StatusCode"] = context.Response.StatusCode.ToString(),
                ["ElapsedMilliseconds"] = stopwatch.Elapsed.TotalMilliseconds.ToString("0.###"),
                ["TraceId"] = Activity.Current?.TraceId.ToString() ?? context.TraceIdentifier
            });
        telemetry.Flush();
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

        telemetry.TrackTrace(
            $"HTTP {context.Request.Method} {context.Request.Path.Value} failed",
            SeverityLevel.Error,
            new Dictionary<string, string>
            {
                ["Method"] = context.Request.Method,
                ["Path"] = context.Request.Path.Value ?? string.Empty,
                ["ElapsedMilliseconds"] = stopwatch.Elapsed.TotalMilliseconds.ToString("0.###"),
                ["ExceptionType"] = exception.GetType().FullName ?? exception.GetType().Name,
                ["TraceId"] = Activity.Current?.TraceId.ToString() ?? context.TraceIdentifier
            });
        telemetry.Flush();

        throw;
    }
});

var summaries = new[]
{
    "Freezing", "Bracing", "Chilly", "Cool", "Mild", "Warm", "Balmy", "Hot", "Sweltering", "Scorching"
};

app.MapGet("/", (ILogger<Program> logger, TelemetryClient telemetry) =>
{
    logger.LogInformation("Hello endpoint called at {ObservedAt}", DateTimeOffset.UtcNow);
    telemetry.TrackTrace(
        "Hello endpoint called",
        SeverityLevel.Information,
        new Dictionary<string, string>
        {
            ["ObservedAt"] = DateTimeOffset.UtcNow.ToString("O"),
            ["Endpoint"] = "/"
        });
    telemetry.Flush();

    return Results.Ok(new
    {
        message = "Hello from the Sealog example Web API",
        weatherUrl = "/weatherforecast"
    });
});

app.MapGet("/weatherforecast", (ILogger<Program> logger, TelemetryClient telemetry) =>
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

    telemetry.TrackTrace(
        "Generated weather forecasts",
        SeverityLevel.Information,
        new Dictionary<string, string>
        {
            ["ForecastCount"] = forecast.Length.ToString(),
            ["MinimumTemperatureC"] = forecast.Min(item => item.TemperatureC).ToString(),
            ["MaximumTemperatureC"] = forecast.Max(item => item.TemperatureC).ToString(),
            ["Endpoint"] = "/weatherforecast"
        });
    telemetry.Flush();

    return forecast;
})
.WithName("GetWeatherForecast");

app.Run();

record WeatherForecast(DateOnly Date, int TemperatureC, string? Summary)
{
    public int TemperatureF => 32 + (int)(TemperatureC / 0.5556);
}
