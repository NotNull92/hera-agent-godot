extends SceneTree

func _initialize() -> void:
	var output: Array = []
	var code := OS.execute(OS.get_executable_path(), PackedStringArray(["--headless", "--path", ProjectSettings.globalize_path("res://"), "--script", "res://tests/headless/mutation_exception_fixture.gd"]), output, true)
	var text := "".join(output)
	if code != 0 or not text.contains("MUTATION_EXCEPTION_PASS") or text.count("intentional gate exception") != 1 or text.contains("leaked at exit"):
		push_error("Exception recovery failed: " + text)
		quit(1)
		return
	print("mutation_exception_test: PASS (one isolated intentional script assertion)")
	quit(0)
