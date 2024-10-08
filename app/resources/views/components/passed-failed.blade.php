@props(['passed' => 0, 'failed' => 0, 'showZero' => false])

@if ($passed > 0 || $showZero)
    <span class="pass">{{ $passed }} passed</span>
@endif

@if (($passed > 0 && $failed > 0) || $showZero)
    and
@endif

@if ($failed > 0 || $showZero)
    <span class="fail">{{ $failed }} failed</span>
@endif

@if ($passed == 0 && $failed == 0 && !$showZero)
    <span class="no-runs">None</span>
@endif
