@php
    /** @var \Illuminate\Support\Collection $activity */
@endphp

<div class="activity-feed">
    @foreach ($activity as $entry)
        @php
            /** @var App\Models\SessionActivity $entry */
        @endphp
        <div class="activity-feed-item activity-feed--{{ $entry->type }}">
            <div class="activity-feed-header">
                @switch($entry->type)
                    @case(\App\Enums\SessionActivityType::review_requested)
                        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"
                            stroke-width="1.5" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round"
                                d="M9.879 7.519c1.171-1.025 3.071-1.025 4.242 0 1.172 1.025 1.172 2.687 0 3.712-.203.179-.43.326-.67.442-.745.361-1.45.999-1.45 1.827v.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 5.25h.008v.008H12v-.008Z" />
                        </svg>
                    @break

                    @case(\App\Enums\SessionActivityType::review_approved)
                        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"
                            stroke-width="1.5" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round"
                                d="M9 12.75 11.25 15 15 9.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
                        </svg>
                    @break

                    @case(\App\Enums\SessionActivityType::review_rejected)
                        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"
                            stroke-width="1.5" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round"
                                d="m9.75 9.75 4.5 4.5m0-4.5-4.5 4.5M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
                        </svg>
                    @break

                    @case(\App\Enums\SessionActivityType::session_created)
                        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"
                            stroke-width="1.5" stroke="currentColor">
                            <circle cx="12" cy="12" r="9" />
                            <path stroke-linecap="round" stroke-linejoin="round" d="M12 8v8m4-4H8" />
                        </svg>
                    @break

                    @case(\App\Enums\SessionActivityType::run_dispatched)
                        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"
                            stroke-width="1.5" stroke="currentColor">
                            <circle cx="12" cy="12" r="9" />
                            <path stroke-linecap="round" stroke-linejoin="round" d="M10 8l6 4-6 4V8z" />
                        </svg>
                    @break

                    @case(\App\Enums\SessionActivityType::comment)
                        <strong>{{ $entry->user->name }}</strong> commented
                        <span class="activity-feed-header-timestamp">
                            {{ $entry->created_at->diffForHumans() }}
                        </span>
                    @break
                @endswitch
            </div>

            <div class="activity-feed-body">
                @if ($entry->type === \App\Enums\SessionActivityType::comment)
                    <x-markdown>{{ $entry->body }}</x-markdown>
                @else
                    {{ $entry->body }}
                @endif

                <span
                    class="activity-feed-body-timestamp">{{ $entry->created_at->diffForHumans() }}</span>
            </div>

        </div>
    @endforeach
</div>
