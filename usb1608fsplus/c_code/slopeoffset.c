#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <fcntl.h>
#include <ctype.h>
#include <math.h>

double volts_USB1608FS_Plus(uint16_t value, uint8_t range)
{
  double volt = 0.0;
  switch(range) {
    case 0:   volt = (value - 0x8000)*10.0/32768.; break;
    case 1:    volt = (value - 0x8000)*5.0/32768.; break;
    case 2:  volt = (value - 0x8000)*2.5/32768.; break;
    case 3:    volt = (value - 0x8000)*2.0/32768.; break;
    case 4: volt = (value - 0x8000)*1.25/32768.; break;
    case 5:    volt = (value - 0x8000)*1.0/32768.; break;
    case 6:  volt = (value - 0x8000)*0.625/32768.; break;
    case 7: volt = (value - 0x8000)*0.3125/32768.; break;
    default: printf("Unknown range.\n"); break;
  }
  return volt;
}

int main ()
{
  float slope;
  float offset;
  uint16_t value;
  uint16_t adjvalue;
  uint8_t range;

  value = 0x8000;
  slope = 1.155244;
  offset = -5451.133301;
  range = 3;
  adjvalue = rint(value*slope + offset);
  printf("Value = %#x / Adjusted Value = %#x\n", value, adjvalue);
  printf("Votlage = %lf / Adjusted Voltage = %lf\n",
      volts_USB1608FS_Plus(value, range), volts_USB1608FS_Plus(adjvalue, range));
  printf("value * slope = %f\n", value*slope);
  printf("value * slope - offset = %f\n", value*slope+offset);
  printf("rint(value * slope - offset) = %f\n", rint(value*slope+offset));
  return 0;
}
