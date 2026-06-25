extends RefCounted

const MAX_SAMPLES := 8192

static func analyze(image: Image) -> Dictionary:
	var width := image.get_width()
	var height := image.get_height()
	var total := width * height
	var step := maxi(1, int(ceil(sqrt(float(total) / float(MAX_SAMPLES)))))
	var unique := {}
	var samples := 0
	var brightness_sum := 0.0
	var alpha_sum := 0.0
	for y in range(0, height, step):
		for x in range(0, width, step):
			var color := image.get_pixel(x, y)
			unique[str(color.to_rgba32())] = true
			brightness_sum += (color.r + color.g + color.b) / 3.0
			alpha_sum += color.a
			samples += 1
	var brightness := 0.0 if samples == 0 else brightness_sum / float(samples)
	return {
		"width": width,
		"height": height,
		"samples": samples,
		"nonblank": unique.size() > 1 and alpha_sum > 0.01,
		"unique_colors": unique.size(),
		"brightness_mean": brightness,
	}
