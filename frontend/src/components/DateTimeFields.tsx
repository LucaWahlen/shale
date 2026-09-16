import { parseDate, parseTime, Time } from "@internationalized/date";
import type { DateValue } from "@internationalized/date";
import { Calendar, DateField, DatePicker, Label, TimeField } from "@heroui/react";

interface ShaleDatePickerProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  name?: string;
}

export function ShaleDatePicker({ label, value, onChange, name }: ShaleDatePickerProps) {
  const dateValue: DateValue | null = value === "" ? null : parseDate(value);
  return (
    <DatePicker
      className="w-full"
      name={name}
      value={dateValue}
      onChange={(v) => onChange(v ? `${v.year}-${String(v.month).padStart(2, "0")}-${String(v.day).padStart(2, "0")}` : "")}
    >
      <Label>{label}</Label>
      <DateField.Group fullWidth>
        <DateField.Input>{(segment) => <DateField.Segment segment={segment} />}</DateField.Input>
        <DateField.Suffix>
          <DatePicker.Trigger aria-label={label}>
            <DatePicker.TriggerIndicator />
          </DatePicker.Trigger>
        </DateField.Suffix>
      </DateField.Group>
      <DatePicker.Popover>
        <Calendar aria-label={label}>
          <Calendar.Header>
            <Calendar.YearPickerTrigger>
              <Calendar.YearPickerTriggerHeading />
              <Calendar.YearPickerTriggerIndicator />
            </Calendar.YearPickerTrigger>
            <Calendar.NavButton slot="previous" />
            <Calendar.NavButton slot="next" />
          </Calendar.Header>
          <Calendar.Grid>
            <Calendar.GridHeader>
              {(day) => <Calendar.HeaderCell>{day}</Calendar.HeaderCell>}
            </Calendar.GridHeader>
            <Calendar.GridBody>{(date) => <Calendar.Cell date={date} />}</Calendar.GridBody>
          </Calendar.Grid>
          <Calendar.YearPickerGrid>
            <Calendar.YearPickerGridBody>
              {({ year }) => <Calendar.YearPickerCell year={year} />}
            </Calendar.YearPickerGridBody>
          </Calendar.YearPickerGrid>
        </Calendar>
      </DatePicker.Popover>
    </DatePicker>
  );
}

interface ShaleTimeFieldProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  name?: string;
}

export function ShaleTimeField({ label, value, onChange, name }: ShaleTimeFieldProps) {
  const timeValue: Time | null = value === "" ? null : parseTime(value);
  return (
    <TimeField
      className="w-full"
      name={name}
      granularity="minute"
      hideTimeZone
      value={timeValue}
      onChange={(v) => onChange(v ? `${String(v.hour).padStart(2, "0")}:${String(v.minute).padStart(2, "0")}` : "")}
    >
      <Label>{label}</Label>
      <TimeField.Group fullWidth>
        <TimeField.Input>{(segment) => <TimeField.Segment segment={segment} />}</TimeField.Input>
      </TimeField.Group>
    </TimeField>
  );
}
