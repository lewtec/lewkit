#include <stdint.h>
#include <stdlib.h>
#include <string.h>

#include <glslang/Include/glslang_c_interface.h>
#include <glslang/Public/resource_limits_c.h>

static char last_err[4096];

static void set_err(const char *a, const char *b) {
	size_t n = 0;
	last_err[0] = 0;
	if (a) {
		n = strlen(a);
		if (n >= sizeof(last_err)) {
			n = sizeof(last_err) - 1;
		}
		memcpy(last_err, a, n);
		last_err[n] = 0;
	}
	if (b && n + 1 < sizeof(last_err)) {
		last_err[n] = '\n';
		size_t m = strlen(b);
		if (n + 1 + m >= sizeof(last_err)) {
			m = sizeof(last_err) - 2 - n;
		}
		memcpy(last_err + n + 1, b, m);
		last_err[n + 1 + m] = 0;
	}
}

const char *last_error(void) { return last_err; }

// compile_stage compiles Vulkan GLSL to SPIR-V.
// stage is a glslang_stage_t value: vertex 0, fragment 4, compute 5.
// On success writes a malloc'd buffer to *out_ptr and length to *out_len.
int compile_stage(int stage, const char *src, uint32_t src_len, uint32_t *out_ptr, uint32_t *out_len) {
	(void)src_len;
	last_err[0] = 0;
	if (!src || !out_ptr || !out_len) {
		set_err("nil argument", NULL);
		return 1;
	}
	if (stage != GLSLANG_STAGE_VERTEX && stage != GLSLANG_STAGE_FRAGMENT && stage != GLSLANG_STAGE_COMPUTE) {
		set_err("stage", NULL);
		return 1;
	}
	*out_ptr = 0;
	*out_len = 0;
	if (!glslang_initialize_process()) {
		set_err("glslang_initialize_process", NULL);
		return 1;
	}
	glslang_input_t input = {
		.language = GLSLANG_SOURCE_GLSL,
		.stage = (glslang_stage_t)stage,
		.client = GLSLANG_CLIENT_VULKAN,
		.client_version = GLSLANG_TARGET_VULKAN_1_1,
		.target_language = GLSLANG_TARGET_SPV,
		.target_language_version = GLSLANG_TARGET_SPV_1_3,
		.code = src,
		.default_version = 450,
		.default_profile = GLSLANG_NO_PROFILE,
		.force_default_version_and_profile = 0,
		.forward_compatible = 0,
		.messages = GLSLANG_MSG_DEFAULT_BIT | GLSLANG_MSG_SPV_RULES_BIT | GLSLANG_MSG_VULKAN_RULES_BIT,
		.resource = glslang_default_resource(),
	};
	glslang_shader_t *shader = glslang_shader_create(&input);
	if (!shader) {
		set_err("shader_create", NULL);
		glslang_finalize_process();
		return 1;
	}
	if (!glslang_shader_preprocess(shader, &input)) {
		set_err("preprocess", glslang_shader_get_info_log(shader));
		glslang_shader_delete(shader);
		glslang_finalize_process();
		return 1;
	}
	if (!glslang_shader_parse(shader, &input)) {
		set_err("parse", glslang_shader_get_info_log(shader));
		glslang_shader_delete(shader);
		glslang_finalize_process();
		return 1;
	}
	glslang_program_t *program = glslang_program_create();
	if (!program) {
		set_err("program_create", NULL);
		glslang_shader_delete(shader);
		glslang_finalize_process();
		return 1;
	}
	glslang_program_add_shader(program, shader);
	if (!glslang_program_link(program, input.messages)) {
		set_err("link", glslang_program_get_info_log(program));
		glslang_program_delete(program);
		glslang_shader_delete(shader);
		glslang_finalize_process();
		return 1;
	}
	glslang_program_SPIRV_generate(program, (glslang_stage_t)stage);
	size_t words = glslang_program_SPIRV_get_size(program);
	if (words == 0) {
		const char *msg = glslang_program_SPIRV_get_messages(program);
		set_err("spirv", msg);
		glslang_program_delete(program);
		glslang_shader_delete(shader);
		glslang_finalize_process();
		return 1;
	}
	size_t nbytes = words * 4;
	unsigned int *buf = (unsigned int *)malloc(nbytes);
	if (!buf) {
		set_err("oom", NULL);
		glslang_program_delete(program);
		glslang_shader_delete(shader);
		glslang_finalize_process();
		return 1;
	}
	glslang_program_SPIRV_get(program, buf);
	*out_ptr = (uint32_t)(uintptr_t)buf;
	*out_len = (uint32_t)nbytes;
	glslang_program_delete(program);
	glslang_shader_delete(shader);
	glslang_finalize_process();
	return 0;
}

int compile_compute(const char *src, uint32_t src_len, uint32_t *out_ptr, uint32_t *out_len) {
	return compile_stage(GLSLANG_STAGE_COMPUTE, src, src_len, out_ptr, out_len);
}
