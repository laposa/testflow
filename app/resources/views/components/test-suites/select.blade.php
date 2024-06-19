@props(['select' => true, 'tests' => []])
<table @class([])>
    <thead>
        <tr>
            <th colspan="2">Select tests</th>
            <th></th>
        </tr>
    </thead>
    <tbody>
        @foreach($tests as $service)
        @foreach($service['children'] as $suite)
        <tr>
            @if ($select)
            <td>
                <div class="suite-selector" data-value="{{ json_encode($service) }}">
                    <label for="selector-suite-{{ $suite['path'] }}" class="checkbox suite">
                        <input type="checkbox" id="selector-suite-{{ $suite['path'] }}">
                        <span class="checkmark"></span>
                    </label>
                </div>
            </td>
            @endif
            <td>{{$suite['full_path']}}</td>
            <td><span class="expand" data-target="selector-suite-tests-{{ $suite['full_path'] }}">Expand</span></td>
        </tr>
        <tr id="selector-suite-tests-{{ $suite['full_path'] }}" class="collapsible">
            <td colspan="3">
                <table>
                    @foreach ($suite['children'] as $test)
                    <tr>
                        <td>
                            <div class="test-selector" data-value="{{ json_encode($suite) }}">
                                <label for="selector-suite-{{ $suite['path'] }}" class="checkbox suite">
                                    <input type="checkbox" id="selector-suite-{{ $suite['path'] }}">
                                    <span class="checkmark"></span>
                                </label>
                            </div>
                        </td>
                        <td>{{$test['name']}}</td>
                    </tr>
                    @endforeach
                </table>
            </td>
        </tr>
        @endforeach
        @endforeach
    </tbody>
</table>


<script>
    document.querySelectorAll(".suite-selector").forEach((selector) => {
        var checkbox = selector.querySelector('input[type="checkbox"]');
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