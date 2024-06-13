<x-layout>
    @php
        $tools = [
            [
                "title" => "Sessions",
                "description" => "Management of testing sessions",
                "link" => "/sessions",
            ],
            [
                "title" => "Tests",
                "description" => "List of all tests and their results",
                "link" => "/tests",
            ],
            [
                "title" => "Runs",
                "description" => "List of previous runs",
                "link" => "/runs",
            ],
            [
                "title" => "Maintenance",
                "description" => "Overview online service health including status of last sync jobs execution",
                "link" => "/maintenance",
            ]
        ]
    @endphp

        <x-portal-section width="full">
            <x-tile-list :tiles="$tools" />
        </x-portal-section>

</x-layout>
