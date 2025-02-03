@props(['passed' => 0, 'failed' => 0, 'skipped' => 0, 'showZero' => false])

@if ($passed > 0 || $showZero)
    <span class="pass">{{ $passed }} passed</span>
@endif

@if ($showZero || ($passed > 0 && ($skipped > 0 && $failed > 0)))
    ,
@elseif ($passed > 0 && ($skipped > 0 || $failed > 0))
    and
@endif

@if ($failed > 0 || $showZero)
    <span class="fail">{{ $failed }} failed</span>
@endif

@if ($showZero || ($skipped > 0 && ($passed > 0 || $failed > 0)))
    and
@endif

@if ($skipped > 0 || $showZero)
    <span class="skip">{{ $skipped ? $skipped : 0 }} skipped</span>
@endif

@if ($passed == 0 && $failed == 0 && $skipped == 0 && !$showZero)
    <span class="no-runs">None</span>
@endif
