@php
/**
 * @var \App\Models\Session $session
 * @var \Illuminate\Support\Collection $activity
 * @var \App\Models\ReviewRequest $reviewRequest
 * @var \App\Models\SessionRun $latestRun
 */
@endphp
<x-layout>
    <x-portal-section title="{{ $session->name }}">
        <livewire:runs.list-runs :session="$session" />
    </x-portal-section>

    @if ($activity && $activity->count() > 0)
        <div class="activity-feed">
            @foreach ($activity as $entry)
                <livewire:activity.activity-feed-item :entry="$entry" />
            @endforeach
        </div>

        @if ($reviewRequest)
            <div>
                <livewire:reviews.review-form :review-request="$reviewRequest" :model="$latestRun" />
            </div>
        @endif
    @endif

    @if ($latestRun)
        <div>
            <livewire:comments.create-comment :model="$latestRun" />
        </div>
        <div >
            <livewire:reviews.request-review :model="$latestRun" />
        </div>
    @endif



    <x-portal-section title="Selected Tests">
        <ul class="session-selected-suites">
            @foreach ($session->itemsGrouped as $path => $tests)
                <li><b>{{ getTestSuiteName($tests[0]) }}</b></li>
                <ul>
                    @foreach ($tests as $test)
                        <li>{{ $test['test_name'] }}</li>
                    @endforeach
                </ul>
            @endforeach
        </ul>
    </x-portal-section>
</x-layout>
