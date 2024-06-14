<x-layout>
    @php
        $tools = [
            [
                "title" => "Sessions",
                "description" => "Management of testing sessions",
                "link" => "/sessions",
            ]
        ]
    @endphp

        <x-portal-section width="full">
            <x-tile-list :tiles="$tools" />
        </x-portal-section>

</x-layout>
