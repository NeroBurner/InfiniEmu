#pragma once

#include <stdint.h>
#include <capstone/capstone.h>

#include "arm.h"
#include "memory.h"

typedef struct cpu_inst_t cpu_t;

cpu_t *cpu_new(uint8_t *program, size_t program_size, memreg_t *mem, size_t max_external_interrupts);
void cpu_free(cpu_t *cpu);
void cpu_reset(cpu_t *cpu);
void cpu_step(cpu_t *cpu);

bool cpu_mem_read(cpu_t *cpu, uint32_t addr, uint8_t *value);
bool cpu_mem_write(cpu_t *cpu, uint32_t addr, uint8_t value);

uint32_t cpu_reg_read(cpu_t *cpu, arm_reg reg);
void cpu_reg_write(cpu_t *cpu, arm_reg reg, uint32_t value);

uint32_t cpu_sysreg_read(cpu_t *cpu, arm_sysreg reg);
void cpu_sysreg_write(cpu_t *cpu, arm_sysreg reg, uint32_t value);

void cpu_jump_exception(cpu_t *cpu, arm_exception ex);
int16_t cpu_get_exception_priority(cpu_t *cpu, arm_exception ex);
void cpu_set_exception_priority(cpu_t *cpu, arm_exception ex, int16_t priority);
