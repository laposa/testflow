@props(['select' => true, 'suites' => []])
<table @class([])>
    <thead>
    <tr>
        @if ($select)
        <th>
        </th>
        @endif

            <th>Test suite</th>
            <th>Workflow</th>
        <th class="center">Last result<br>Pass/Fail</th>
        @endif
        <th></th>
    </tr>
    </thead>
    <tbody>
        @foreach($suites as $suite)
        <tr>
            @if ($select)
            <td>
                <div class="suite-selector" data-value="{{ json_encode($suite) }}">
                    <label for="selector-suite-{{ $suite['path'] }}" class="checkbox suite">
                        <input type="checkbox" id="selector-suite-{{ $suite['path'] }}">
                        <span class="checkmark"></span>
                    </label>
                </div>
            </td>
            @endif
                <td>{{$suite['path']}}</td>
                <td>{{$suite['workflow']['name']}}</td>
            <td class="center"><span class="pass">1</span>/<span class="fail">1</span></td>
            <td><span class="expand" data-target="selector-suite-tests-{{ $suite['repository_id'] }}">Expand</span></td>
        </tr>
        <tr id="selector-suite-tests-{{ $suite['repository_id'] }}" class="collapsible">
            <td>
            </td>
        </tr>
        @endforeach
    </tbody>
</table>


<script>
    document.querySelectorAll(".suite-selector").forEach((selector) => {
        var checkbox = selector.querySelector('input[type="checkbox"]');
        console.log(checkbox);
        checkbox.addEventListener("change", (e) => {
            if (checkbox.checked) {
                const el = document.createElement('input');
                const value = JSON.parse(selector.dataset.value);
                el.type = 'hidden';
                el.value = JSON.stringify(value);
                el.name = `data[${value.path}]`;
                selector.appendChild(el);
            } else {
                selector.querySelector('input[type="hidden"]').remove();
            }
        })
    });
</script>

