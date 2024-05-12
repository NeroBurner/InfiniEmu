#pragma once

#include "cpu.h"
#include "bus_i2c.h"
#include "bus_spi.h"

#define NRF52832_SRAM_SIZE 0x10000
#define NRF52832_FLASH_SIZE 0x80000

#define NRF52832_PRIORITY_BITS 3

#define NRF52832_MAX_EXTERNAL_INTERRUPTS 496

typedef struct NRF52832_inst_t NRF52832_t;

NRF52832_t *nrf52832_new(uint8_t *program, size_t program_size);
void nrf52832_reset(NRF52832_t *nrf52832);
void nrf52832_step(NRF52832_t *nrf52832);

cpu_t *nrf52832_get_cpu(NRF52832_t *);
bus_spi_t *nrf52832_get_spi(NRF52832_t *);
bus_i2c_t *nrf52832_get_i2c(NRF52832_t *);
