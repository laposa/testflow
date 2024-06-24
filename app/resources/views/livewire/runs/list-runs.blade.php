<div wire:poll.5s>
    @foreach($session->itemsGrouped as $path => $tests)
    <x-portal-section title="{{ getTestSuiteName($tests[0]) }}" width="full">
        <x-runs.list :runs="$tests[0]->runs" />
    </x-portal-section>
    @endforeach
</div>