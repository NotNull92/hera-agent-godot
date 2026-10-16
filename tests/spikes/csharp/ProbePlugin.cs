#if TOOLS
using Godot;
using System;
using System.IO;
using System.Linq;
using System.Reflection;
using System.Runtime.CompilerServices;
using System.Runtime.Loader;

[Tool]
public partial class ProbePlugin : EditorPlugin
{
    private string _directory = "";
    private WeakReference? _context;
    private int _attempts;

    public override void _EnterTree()
    {
        _directory = OS.GetEnvironment("HERA_SPIKE_DIRECTORY");
        File.WriteAllText(Path.Combine(_directory, "loaded-" + ProbeStamp.Value), "loaded");
        SetProcess(true);
    }

    public override void _Process(double delta)
    {
        // Reload restores an existing node; it need not call _EnterTree again.
        _directory = OS.GetEnvironment("HERA_SPIKE_DIRECTORY");
        string marker = Path.Combine(_directory, "loaded-" + ProbeStamp.Value);
        if (!File.Exists(marker)) File.WriteAllText(marker, "processing");
        if (File.Exists(Path.Combine(_directory, "quit")))
        {
            GetTree().Quit();
            return;
        }
        string request = Path.Combine(_directory, "eval-request");
        if (File.Exists(request))
        {
            File.Delete(request);
            try
            {
                _context = Execute(Path.Combine(_directory, "expression.dll"));
                _attempts = 0;
            }
            catch (Exception e)
            {
                File.WriteAllText(Path.Combine(_directory, "failed"), e.ToString());
            }
        }
        if (_context != null)
        {
            GC.Collect();
            GC.WaitForPendingFinalizers();
            GC.Collect();
            if (!_context.IsAlive)
            {
                File.WriteAllText(Path.Combine(_directory, "unloaded-" + ProbeStamp.Value), "collected");
                _context = null;
            }
            else if (++_attempts >= 60)
            {
                File.WriteAllText(Path.Combine(_directory, "failed"), "ALC remained alive after 60 GC cycles");
                _context = null;
            }
        }
    }

    [MethodImpl(MethodImplOptions.NoInlining)]
    private WeakReference Execute(string path)
    {
        var context = new EvaluationContext();
        var weak = new WeakReference(context);
        try
        {
            using var stream = File.OpenRead(path);
            var assembly = context.LoadFromStream(stream);
            var root = new Node2D { Position = new Vector2(12, 34) };
            try
            {
                object? value = assembly.GetType("Expression")!.GetMethod("Run")!.Invoke(null, new object[] { root });
                using Variant result = Variant.From((Vector2)value!);
                if (result.AsVector2() != new Vector2(12, 34))
                    throw new InvalidOperationException("Engine API result mismatch");
                File.WriteAllText(Path.Combine(_directory, "result-" + ProbeStamp.Value), result.ToString());
            }
            finally
            {
                root.Free();
            }
        }
        finally
        {
            context.Unload();
        }
        return weak;
    }

    private sealed class EvaluationContext : AssemblyLoadContext
    {
        public EvaluationContext() : base(isCollectible: true) { }
        protected override Assembly? Load(AssemblyName name) =>
            AppDomain.CurrentDomain.GetAssemblies().FirstOrDefault(a =>
                a != null && a.GetName().Name == name.Name && GetLoadContext(a) != this);
    }
}
#endif
