using System.Text.Json;
using Microsoft.CodeAnalysis;
using Microsoft.CodeAnalysis.CSharp;
using System.Reflection.Metadata;
using System.Reflection.PortableExecutable;

try
{
    if (args.Length != 2 || args[0] != "--request") throw new ArgumentException("expected --request FILE");
    var file = new FileInfo(args[1]);
    if (file.Length > 1024 * 1024) throw new ArgumentException("compile request exceeds 1 MiB");
    var request = JsonSerializer.Deserialize<CompileRequest>(File.ReadAllText(file.FullName))!;
    if (string.IsNullOrWhiteSpace(request.expression) || request.expression.Length > 65536)
        throw new ArgumentException("expression must contain 1–65536 characters");
    if (request.references is null || request.references.Length > 256)
        throw new ArgumentException("invalid assembly references");
    MetadataReference? projectReference = null;
    if (request.project_reference != null)
    {
        var projectBytes = File.ReadAllBytes(request.project_reference);
        using var project = new MemoryStream(projectBytes, writable: false);
        using var pe = new PEReader(project);
        var metadata = pe.GetMetadataReader();
        if (metadata.GetGuid(metadata.GetModuleDefinition().Mvid).ToString() != request.project_mvid)
            throw new ArgumentException("project assembly on disk differs from the editor; rebuild/reload before eval");
        projectReference = MetadataReference.CreateFromImage(projectBytes);
    }
    var expression = SyntaxFactory.ParseExpression(request.expression, consumeFullText: true);
    if (expression.ContainsDiagnostics)
    {
        Failure("parse_error", expression.GetDiagnostics().Select(d => d.ToString()));
        return 1;
    }
    var paths = ((string?)AppContext.GetData("TRUSTED_PLATFORM_ASSEMBLIES") ?? "").Split(Path.PathSeparator)
        .Concat(request.references).Where(p => p.Length > 0)
        .GroupBy(Path.GetFileName, StringComparer.OrdinalIgnoreCase).Select(g => g.Last());
    var references = paths.Select(p => projectReference != null && Path.GetFullPath(p) == Path.GetFullPath(request.project_reference!)
        ? projectReference : MetadataReference.CreateFromFile(Path.GetFullPath(p))).ToArray();
    var prefix = "using System; using System.Linq; using Godot; public static class HeraExpression { public static object? Run(Node SceneRoot) {\n";
    var source = prefix + "#line 1 \"expression\"\nreturn (object?)(" + request.expression + ");\n} }";
    var compilation = Compile(source, references);
    using var output = new MemoryStream();
    var emitted = compilation.Emit(output);
    if (!emitted.Success)
    {
        // A void-returning invocation is still a valid expression with a nil result.
        using var voidOutput = new MemoryStream();
        var voidEmission = Compile(prefix + "#line 1 \"expression\"\n" + request.expression + "; return null;\n} }", references).Emit(voidOutput);
        if (!voidEmission.Success)
        {
            Failure("parse_error", emitted.Diagnostics.Where(d => d.Severity == DiagnosticSeverity.Error).Select(d => d.ToString()));
            return 1;
        }
        output.SetLength(0);
        voidOutput.Position = 0;
        voidOutput.CopyTo(output);
    }
    if (output.Length > 384 * 1024) throw new ArgumentException("compiled expression exceeds 384 KiB");
    Console.WriteLine(JsonSerializer.Serialize(new { ok = true, assembly = Convert.ToBase64String(output.ToArray()) }));
    return 0;
}
catch (Exception e)
{
    Failure("compile_failed", new[] { e.Message });
    return 1;
}

static CSharpCompilation Compile(string source, MetadataReference[] references) =>
    CSharpCompilation.Create("HeraEval_" + Guid.NewGuid().ToString("N"),
        new[] { CSharpSyntaxTree.ParseText(source) }, references,
        new CSharpCompilationOptions(OutputKind.DynamicallyLinkedLibrary, optimizationLevel: OptimizationLevel.Release));

static void Failure(string code, IEnumerable<string> diagnostics) =>
    Console.WriteLine(JsonSerializer.Serialize(new { ok = false, error = code, diagnostics = diagnostics.Take(32).ToArray() }));

record CompileRequest(string expression, string[] references, string? project_reference, string? project_mvid);
