@php
    /** @var \App\Models\SessionRun $session */
@endphp

<div @if ($pollingEnabled) wire:poll.2.5s @endif>
    @foreach ($session->itemsGrouped as $path => $tests)
        <x-portal-section title="{{ getTestSuiteName($tests[0]) }}" width="full">
            <x-runs.list :runs="$tests[0]->runs" />
        </x-portal-section>
    @endforeach

    <button type="submit" wire:click="createRun" wire:loading.attr="disabled">Run all tests</button>
</div>
