import { TrafficRange, trafficRanges } from "@lib/adminClient";
import Segmented from "./segmented";

interface RangePickerProps {
  tab: string;
  range: TrafficRange;
}

export default function RangePicker({ tab, range }: RangePickerProps) {
  return (
    <Segmented
      active={range}
      items={trafficRanges.map((r) => ({
        key: r,
        href: `/admin?tab=${tab}&range=${r}`,
        label: r,
      }))}
    />
  );
}
