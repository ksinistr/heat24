# heat24

Visualises daily heat stress by hour for a set of locations, to pick safe times for outdoor activity.

## Language

**Report**:
One of three views over hourly weather: Last Week, Month, Annual. Each report has its own set of locations.
_Avoid_: chart type, mode, period

**Last Week report**:
Per-location 24h curve of the Hybrid Index averaged over the previous 7 days, filled by Risk Level, with Sunrise/Sunset.

**Month report**:
For each configured month of the previous calendar year, one 24h chart overlaying the monthly-averaged Hybrid Index of every location in the set, with Risk Level bands and a list of Comfortable Locations.

**Annual report**:
Per-location heatmap of the Hybrid Index averaged by hour of day × month over the previous calendar year.

**Heat Index**:
NOAA apparent temperature from air temperature and relative humidity.

**Hybrid Index**:
Air temperature below 27 °C, Heat Index at or above 27 °C. The single value every report displays.
The switch point is the upper bound of the Suitable Risk Level, not an independent constant.
_Avoid_: HI (ambiguous), feels-like

**Risk Level**:
NOAA band of the Hybrid Index: Suitable (<27), Caution (27–32), Extreme Caution (32–41), Danger (41–54), Extreme Danger (≥54).

**Comfortable Location**:
In a Month report, a location with at least one hour of data where every hour with data is Suitable. A location that failed to load is never comfortable.

**Location**:
A named Point weather is fetched for. The name belongs to reports only.

**Point**:
Latitude and longitude. Weather data, including its cache entry, is identified by Point, never by Location name.
