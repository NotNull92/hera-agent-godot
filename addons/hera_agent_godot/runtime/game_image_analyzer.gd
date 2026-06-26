extends RefCounted

const MAX_SAMPLES := 8192
const EDGE_BAND_RATIO := 0.025
const EDGE_CONTENT_THRESHOLD := 0.08
const COLOR_DELTA_THRESHOLD := 0.08
const DARK_BRIGHTNESS_THRESHOLD := 0.08

static func analyze(image: Image) -> Dictionary:
	var width := image.get_width()
	var height := image.get_height()
	var total := width * height
	var step := maxi(1, int(ceil(sqrt(float(total) / float(MAX_SAMPLES)))))
	var edge_band_x := maxi(1, int(ceil(float(width) * EDGE_BAND_RATIO)))
	var edge_band_y := maxi(1, int(ceil(float(height) * EDGE_BAND_RATIO)))
	var background_color := image.get_pixel(0, 0)
	var unique := {}
	var samples := 0
	var brightness_sum := 0.0
	var alpha_sum := 0.0
	var dark_samples := 0
	var edge_samples := 0
	var edge_content_samples := 0
	for y in range(0, height, step):
		for x in range(0, width, step):
			var color := image.get_pixel(x, y)
			var brightness_value := (color.r + color.g + color.b) / 3.0
			unique[str(color.to_rgba32())] = true
			brightness_sum += brightness_value
			alpha_sum += color.a
			if brightness_value <= DARK_BRIGHTNESS_THRESHOLD:
				dark_samples += 1
			if _is_edge_sample(x, y, width, height, edge_band_x, edge_band_y):
				edge_samples += 1
				if _color_delta(color, background_color) >= COLOR_DELTA_THRESHOLD:
					edge_content_samples += 1
			samples += 1
	var brightness := 0.0 if samples == 0 else brightness_sum / float(samples)
	var alpha := 0.0 if samples == 0 else alpha_sum / float(samples)
	var dark_ratio := 0.0 if samples == 0 else float(dark_samples) / float(samples)
	var edge_content_ratio := 0.0 if edge_samples == 0 else float(edge_content_samples) / float(edge_samples)
	return {
		"width": width,
		"height": height,
		"samples": samples,
		"nonblank": unique.size() > 1 and alpha_sum > 0.01,
		"unique_colors": unique.size(),
		"brightness_mean": brightness,
		"alpha_mean": alpha,
		"dark_sample_ratio": dark_ratio,
		"edge_content_ratio": edge_content_ratio,
		"edge_content_detected": edge_content_ratio >= EDGE_CONTENT_THRESHOLD,
		"possible_clipping": edge_content_ratio >= EDGE_CONTENT_THRESHOLD,
		"low_detail": unique.size() < 4,
	}

static func _is_edge_sample(x: int, y: int, width: int, height: int, edge_band_x: int, edge_band_y: int) -> bool:
	return x < edge_band_x or y < edge_band_y or x >= width - edge_band_x or y >= height - edge_band_y

static func _color_delta(a: Color, b: Color) -> float:
	return (absf(a.r - b.r) + absf(a.g - b.g) + absf(a.b - b.b) + absf(a.a - b.a)) / 4.0
