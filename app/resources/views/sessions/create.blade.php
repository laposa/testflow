<x-layout>
    <x-portal-section title="Create new session" width="full">
        <div class="information">
            <input type="text" id="sessionName" name="sessionName" required placeholder="Session name">
            <select name="assignees" id="">
                <option value="" disabled selected>Assign to</option>
                <option value="1">User 1</option>
                <option value="2">User 2</option>
                <option value="3">User 3</option>
            </select>
        </div>
    </x-portal-section>
    <x-portal-section title="Test suites" width="full">
        <x-test-suites.list :select="true" :suites="$suites"/>
    </x-portal-section>

    <a class="button" href="/sessions/1">Create</a>
</x-layout>
