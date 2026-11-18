# DENSITY entry on a minute pinbar

Live trading and Telegram simulation use the same trigger. The closed minute's low lies between the confirmed bid density and that density plus the configured entry-zone ticks, inclusively. The body is in the upper third and at most 25% of the range. The lower wick spans at least 60% of the range.

Submit one buy during the following minute at the available ask, regardless of its distance from the density. The existing funds and inventory checks apply. Prefer the broker's closed candles to aggregating trades because the extended universe currently uses books without trade subscriptions. Use an immediate-or-cancel limit at the observed ask so that the order cannot rest beyond the signal or use an unchecked amount.

Before submitting, allow an eligible density to move up to two ticks in total from the pinbar level. A loss or a larger move invalidates the signal. After buying, use the existing relocation criteria and keep the protective prices fixed across relocations. Compute the target using the execution price. Trading exits are density loss, stop and target. Keep broker protection during stale market data. Remove the sell-wall target from Telegram simulation.
