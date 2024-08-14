@php
    /** @var App\Models\Comment|App\Models\ReviewRequest $entry */
@endphp

@if ($entry instanceof App\Models\Comment)
    @php
        /** @var App\Models\Comment $entry */
    @endphp
    <div class="activity-feed-item activity-feed-item--comment">
        <div>
            <div class="activity-feed-item-header">
                <div>
                    <strong>{{ $entry->user->name }}</strong> commented
                    {{ $entry->created_at->diffForHumans() }}
                </div>
            </div>
            <div class="activity-feed-item-body">{{ $entry->body }}</div>
        </div>
    </div>
@endif
@if ($entry instanceof \Spatie\Activitylog\Models\Activity)
    @php
        /** @var \Spatie\Activitylog\Models\Activity $entry */
    @endphp
    <div
        class="activity-feed-item activity-feed-item--review-request activity-feed--{{ $entry->event }}">

        @switch($entry->event)
            @case('review_request')
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5"
                    stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round"
                        d="M9.879 7.519c1.171-1.025 3.071-1.025 4.242 0 1.172 1.025 1.172 2.687 0 3.712-.203.179-.43.326-.67.442-.745.361-1.45.999-1.45 1.827v.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 5.25h.008v.008H12v-.008Z" />
                </svg>
            @break

            @case('review_approved')
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"
                    stroke-width="1.5" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round"
                        d="M9 12.75 11.25 15 15 9.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
                </svg>
            @break

            @case('review_rejected')
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"
                    stroke-width="1.5" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round"
                        d="m9.75 9.75 4.5 4.5m0-4.5-4.5 4.5M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
                </svg>
            @break
        @endswitch
        {{ $entry->description }}
        <div>{{ $entry->created_at->diffForHumans() }}</div>
    </div>
@endif
