#if TOOLS
using Godot;
using System;
using System.IO;
using System.Reflection;
using System.Runtime.CompilerServices;
using System.Runtime.Loader;

[Tool]
public partial class HeraCSharpEval : RefCounted
{
    public Godot.Collections.Dictionary CompilationContext()
    {
        var assembly = typeof(HeraCSharpEval).Assembly;
        // Godot loads the project from bytes; its Assembly.Location is empty.
        var project = ProjectSettings.GlobalizePath("res://.godot/mono/temp/bin/Debug/" + assembly.GetName().Name + ".dll");
        return new()
        {
            ["project_reference"] = project,
            ["project_mvid"] = assembly.ManifestModule.ModuleVersionId.ToString(),
            ["references"] = new string[] { typeof(Node).Assembly.Location, typeof(EditorInterface).Assembly.Location, project }
        };
    }

    public Godot.Collections.Dictionary Evaluate(byte[] bytes, Node sceneRoot, string projectMvid)
    {
        if (!Engine.IsEditorHint()) return Failure("csharp_unavailable: C# eval is editor-only");
        if (bytes.Length == 0 || bytes.Length > 384 * 1024) return Failure("invalid C# assembly size");
        if (projectMvid != typeof(HeraCSharpEval).Assembly.ManifestModule.ModuleVersionId.ToString())
            return Failure("state_conflict: project assembly reloaded after expression compilation");
        var (context, response) = Execute(bytes, sceneRoot);
        for (int i = 0; i < 8 && context.IsAlive; i++)
        {
            GC.Collect();
            GC.WaitForPendingFinalizers();
            GC.Collect();
        }
        if (context.IsAlive)
            return Failure("csharp_unload_failed: evaluation retained references; effects are not rolled back and editor reload may be affected");
        return response;
    }

    [MethodImpl(MethodImplOptions.NoInlining)]
    private static (WeakReference, Godot.Collections.Dictionary) Execute(byte[] bytes, Node sceneRoot)
    {
        var context = new EvaluationContext();
        var weak = new WeakReference(context);
        try
        {
            using var stream = new MemoryStream(bytes, writable: false);
            var assembly = context.LoadFromStream(stream);
            var method = assembly.GetType("HeraExpression", throwOnError: true)!.GetMethod("Run", BindingFlags.Public | BindingFlags.Static);
            if (method == null) throw new InvalidOperationException("compiled expression entry point missing");
            var value = method.Invoke(null, new object?[] { sceneRoot });
            return (weak, new Godot.Collections.Dictionary { ["ok"] = true, ["value"] = ToVariant(value) });
        }
        catch (Exception e)
        {
            return (weak, Failure("execute error: " + (e.InnerException ?? e).Message));
        }
        finally
        {
            context.Unload();
        }
    }

    private static Variant ToVariant(object? value) => value switch
    {
        null => default,
        Variant v => v,
        bool v => v, byte v => v, sbyte v => v, short v => v, ushort v => v,
        int v => v, uint v => v, long v => v, float v => v, double v => v,
        string v => v, StringName v => v, NodePath v => v,
        Vector2 v => v, Vector2I v => v, Vector3 v => v, Vector3I v => v,
        Vector4 v => v, Vector4I v => v, Rect2 v => v, Rect2I v => v,
        Transform2D v => v, Transform3D v => v, Basis v => v, Quaternion v => v,
        Projection v => v, Plane v => v, Aabb v => v, Color v => v, Rid v => v,
        Godot.Collections.Dictionary v => v, Godot.Collections.Array v => v,
        byte[] v => v, int[] v => v, long[] v => v, float[] v => v, double[] v => v,
        string[] v => v, Vector2[] v => v, Vector3[] v => v, Vector4[] v => v, Color[] v => v,
        GodotObject v => GodotObject.IsInstanceValid(v) ? Variant.From(v) : default,
        _ => throw new InvalidOperationException("unsupported C# result type: " + value.GetType().FullName + "; return a Godot Variant-compatible value")
    };

    private static Godot.Collections.Dictionary Failure(string error) => new() { ["ok"] = false, ["error"] = error };

    private sealed class EvaluationContext : AssemblyLoadContext
    {
        private readonly AssemblyLoadContext _project = GetLoadContext(typeof(HeraCSharpEval).Assembly)!;
        public EvaluationContext() : base(isCollectible: true) { }
        protected override Assembly? Load(AssemblyName name)
        {
            try { return _project.LoadFromAssemblyName(name); }
            catch (FileNotFoundException) { return null; }
        }
    }
}
#endif
