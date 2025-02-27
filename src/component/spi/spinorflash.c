#include "components/spi/spinorflash.h"

#include <assert.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>

#include "config.h"

#ifdef ENABLE_LOG_SPI_FLASH
#define LOGF(...) printf(__VA_ARGS__)
#else
#define LOGF(...)
#endif

enum
{
    COMMAND_PP = 0x02,    // Page Program
    COMMAND_READ = 0x03,  // Read Data Bytes
    COMMAND_RDSR = 0x05,  // Read Status Register
    COMMAND_WREN = 0x06,  // Write Enable
    COMMAND_SE = 0x20,    // Sector Erase
    COMMAND_RDSER = 0x2B, // Read Security Register - not sure if this is correct
    COMMAND_RDID = 0x9F,  // Read Identification
    COMMAND_RDI = 0xAB,   // Release from Deep Power-Down and Read Device ID
};

#define MAX_COMMAND_SIZE 32

#define READ_UINT24(data, start) (((data)[(start)] << 16) | ((data)[(start) + 1] << 8) | (data)[(start) + 2])

typedef union
{
    struct
    {
        unsigned int WIP : 1;  // Write in progress
        unsigned int WEL : 1;  // Write enable latch
        unsigned int BP0 : 1;  // Block protect 0
        unsigned int BP1 : 1;  // Block protect 1
        unsigned int BP2 : 1;  // Block protect 2
        unsigned int BP3 : 1;  // Block protect 3
        unsigned int BP4 : 1;  // Block protect 4
        unsigned int SRP0 : 1; // Status register protect
        unsigned int SRP1 : 1; // Status register protect
        unsigned int QE : 1;   // Quad enable
        unsigned int LB : 1;   // Security register lock
        unsigned int : 3;
        unsigned int CMP : 1; // Complement protect
    };
    uint16_t value;
} statusreg_t;

typedef union
{
    struct
    {
        // I haven't been able to find information on this register so these fields were extracted from InfiniTime's code

        unsigned int : 4;
        unsigned int PRFAIL : 1; // Program failed
        unsigned int ERFAIL : 1; // Erase failed
    };
    uint8_t value;
} securityreg_t;

typedef struct
{
    uint8_t *data;
    size_t size, sector_size;

    statusreg_t statusreg;
    securityreg_t securityreg;

    uint8_t last_command[MAX_COMMAND_SIZE];
    size_t last_command_size;

    uint32_t pp_address;
} spinorflash_t;

void spinorflash_write(const uint8_t *data, size_t data_size, void *userdata)
{
    if (data_size > MAX_COMMAND_SIZE)
    {
        printf("SPI flash command too long: %zu\n", data_size);
        abort();
    }

#ifdef ENABLE_LOG_SPI_FLASH
    LOGF("SPI flash got data: ");
    for (size_t i = 0; i < data_size; i++)
    {
        LOGF("%02X", data[i]);
        if (i < data_size - 1)
            LOGF("-");
    }
    LOGF("\n");
#endif

    spinorflash_t *flash = (spinorflash_t *)userdata;

    if (flash->statusreg.WIP)
    {
        memcpy(flash->data + flash->pp_address, data, data_size);
        return;
    }

    memcpy(flash->last_command, data, data_size);
    flash->last_command_size = data_size;

    switch (data[0])
    {
    case COMMAND_PP:
    {
        assert(data_size == 4);
        assert(flash->statusreg.WEL);

        uint32_t addr = READ_UINT24(data, 1);
        assert(addr < flash->size);

        flash->pp_address = addr;
        flash->statusreg.WIP = 1;
        break;
    }

    case COMMAND_WREN:
        assert(data_size == 1);

        flash->statusreg.WEL = 1;
        break;

    case COMMAND_SE:
    {
        assert(data_size == 4);
        assert(flash->statusreg.WEL);

        uint32_t addr = READ_UINT24(data, 1);
        assert(addr <= flash->size - flash->sector_size);

        memset(flash->data + addr, 0xFF, flash->sector_size);
        break;
    }

    case COMMAND_READ:
    case COMMAND_RDSR:
    case COMMAND_RDSER:
    case COMMAND_RDID:
    case COMMAND_RDI:
        break; // Handled in read

    default:
        printf("Unknown SPI flash write command: %02X\n", data[0]);
        abort();
    }
}

size_t spinorflash_read(uint8_t *data, size_t data_size, void *userdata)
{
    spinorflash_t *flash = (spinorflash_t *)userdata;

    switch (flash->last_command[0])
    {
    case COMMAND_READ:
    {
        assert(flash->last_command_size == 4);

        uint32_t offset = READ_UINT24(flash->last_command, 1);

        memcpy(data, flash->data + offset, data_size);
        return data_size;
    }

    case COMMAND_RDSR:
        assert(data_size >= 1);

        data[0] = flash->statusreg.value & 0xFF;
        return 1;

    case COMMAND_RDSER:
        assert(data_size >= 1);

        data[0] = flash->securityreg.value;
        return 1;

    case COMMAND_RDID:
        assert(data_size >= 3);

        // Dummy data
        data[0] = 0xA5;
        data[1] = 0xA5;
        data[2] = 0xA5;
        return 3;

    case COMMAND_RDI:
        assert(data_size >= 1);

        // Dummy data
        data[0] = 0xA5;
        return 1;
    }

    printf("Unknown SPI flash command: %02X\n", flash->last_command[0]);
    abort();
}

void spinorflash_reset(void *userdata)
{
    spinorflash_t *flash = (spinorflash_t *)userdata;

    flash->statusreg.value = 0;
    flash->securityreg.value = 0;
    flash->last_command_size = 0;
}

void spinorflash_cs_changed(bool selected, void *userdata)
{
    spinorflash_t *flash = (spinorflash_t *)userdata;

    if (!selected)
    {
        flash->statusreg.WIP = 0;
    }
}

spi_slave_t spinorflash_new(size_t size, size_t sector_size)
{
    spinorflash_t *flash = (spinorflash_t *)malloc(sizeof(spinorflash_t));
    flash->data = (uint8_t *)malloc(size);
    flash->size = size;
    flash->sector_size = sector_size;

    return (spi_slave_t){
        .userdata = flash,
        .read = spinorflash_read,
        .write = spinorflash_write,
        .reset = spinorflash_reset,
        .cs_changed = spinorflash_cs_changed,
    };
}