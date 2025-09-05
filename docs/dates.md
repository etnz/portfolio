## Using Flexible Date Formats

To make entering dates faster and more intuitive, most commands that accept a date flag (`-d`) support several shorthand formats. You can use these formats instead of typing the full `YYYY-MM-DD` date.

To ensure unambiguous parsing, `pcs` attempts to interpret date strings using the following formats:

1.  **Relative Duration Format**

    You can specify a date relative to today using a signed integer and a unit. The sign is mandatory.

    * **Format**: `[sign][number][unit]`
    * **Sign**: `+` for a future date, `-` for a past date.
    * **Unit**: `d` (days), `w` (weeks), `m` (months), `q` (quarters), `y` (years).

    | Example | Assuming Today is 2025-08-29 | Resulting Date |
    | :------ | :--------------------------- | :------------- |
    | `-1d`   | Yesterday                    | `2025-08-28`   |
    | `+1d`   | Tomorrow                     | `2025-08-30`   |
    | `+0d`   | Today                        | `2025-08-29`   |
    | `-2w`   | Two weeks ago                | `2025-08-15`   |
    | `-1m`   | One month ago                | `2025-07-29`   |

2.  **`[MM-]DD` Format**

    You can specify a day, or a month and a day, and the current year will be assumed. This format also has special handling for `0`.

    * **`DD`**: A day in the current month and year.
    * **`MM-DD`**: A specific month and day in the current year.
    * **`0` as the day**: Resolves to the last day of the *previous* month.
    * **`0` as the month**: Resolves to the corresponding day in December of the *previous* year.

    | Example | Assuming Today is 2025-08-29 | Resulting Date |
    | :------ | :--------------------------- | :------------- |
    | `27`    | The 27th of the current month | `2025-08-27`   |
    | `1-15`  | January 15th of current year | `2025-01-15`   |
    | `0`     | Last day of previous month   | `2025-07-31`   |
    | `8-0`   | Last day of July (month 8-1) | `2025-07-31`   |
    | `1-0`   | Last day of previous year    | `2024-12-31`   |
    | `0-15`  | Dec 15th of the previous year | `2024-12-15`   |
    | `0-0`   | Nov 30th of the previous year | `2024-11-30`   |

3.  **`YYYY-MM-DD` Format**

    If the input doesn't match any of the shorthand formats, `pcs` will try to parse it as a full standard date.

    | Example      | Resulting Date |
    | :----------- | :------------- |
    | `2024-02-29` | `2024-02-29`   |
