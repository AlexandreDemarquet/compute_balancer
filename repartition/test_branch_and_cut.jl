using LinearAlgebra

# Fonction pour résoudre la relaxation LP avec Simplex (ou autre méthode LP)
function solve_relaxed_lp(A, b, c)
    # Nombre de contraintes et de variables
    m, n = size(A)

    # Initialisation du tableau Simplex
    # Ajout des variables d'écart
    tableau = hcat(A, I(m), b)  # Ajoute les variables d'écart
    tableau = vcat(tableau, hcat(c, zeros(Int64, 1, m + n), 0))  # Ajoute la ligne de la fonction objectif

    # Indices des variables de base
    base = n + 1:n + m

    while true
        # Calculer les coûts réduits
        c_b = tableau[end, 1:n + m]  # Coefficients de la fonction objectif
        tableau[end, 1:n + m] .-= c_b' * tableau[1:end-1, 1:n + m]

        # Vérifiez si la solution est optimale
        if all(tableau[end, 1:n] .<= 0)  # Tous les coûts réduits doivent être <= 0 pour une maximisation
            break  # La solution est optimale
        end

        # Trouver la variable entrante
        entering_var = argmax(tableau[end, 1:n])  # Choisit la variable avec le coût réduit le plus élevé

        # Calculer les rapports pour déterminer la variable sortante
        ratios = []
        for i in 1:m
            if tableau[i, entering_var] > 0  # Évitez la division par zéro
                push!(ratios, tableau[i, end] / tableau[i, entering_var])
            else
                push!(ratios, Inf)  # Pas de limite
            end
        end

        # Trouver la variable sortante
        leaving_var = argmin(ratios)

        # Mettre à jour les indices de base
        base[leaving_var] = entering_var

        # Mettre à jour le tableau
        pivot = tableau[leaving_var, entering_var]
        tableau[leaving_var, :] ./= pivot  # Normaliser la ligne pivot

        for i in 1:m+1
            if i != leaving_var
                tableau[i, :] .-= tableau[i, entering_var] .* tableau[leaving_var, :]
            end
        end
    end

    # Extraire la solution
    solution = zeros(n)
    for i in 1:m
        if base[i] <= n  # Si la variable de base est une variable originale
            solution[base[i]] = tableau[i, end]
        end
    end

    return solution
end

# Fonction de branchement et de coupe (branch and cut)
function branch_and_cut(A, b, c, variables, depth = 0)
    # Résoudre le problème relaxé
    x_relaxed = solve_relaxed_lp(A, b, c)

    # Vérifier si la solution est entière
    if all(x -> isinteger(x), x_relaxed)
        println("Solution entière trouvée : ", x_relaxed)
        return x_relaxed
    end

    # Si la solution n'est pas entière, on branche sur une variable fractionnaire
    for i in 1:length(x_relaxed)
        if x_relaxed[i] ≠ 0 && x_relaxed[i] ≠ 1
            println("Branching on variable $i at depth $depth")

            # Créer deux sous-problèmes : un avec x[i] = 0, l'autre avec x[i] = 1
            # Branch 1: Ajouter la contrainte x[i] = 0
            new_variables_zero = copy(variables)
            new_variables_zero[i] = (0, 0)
            solution_zero = branch_and_cut(A, b, c, new_variables_zero, depth + 1)

            # Branch 2: Ajouter la contrainte x[i] = 1
            new_variables_one = copy(variables)
            new_variables_one[i] = (1, 1)
            solution_one = branch_and_cut(A, b, c, new_variables_one, depth + 1)

            # Comparer les solutions trouvées dans les deux branches
            if solution_zero !== nothing && solution_one !== nothing
                return norm(c * solution_zero) < norm(c * solution_one) ? solution_zero : solution_one
            elseif solution_zero !== nothing
                return solution_zero
            elseif solution_one !== nothing
                return solution_one
            else
                return nothing  # Aucun résultat valide trouvé
            end
        end
    end
end

function setup_constraints(num_tasks, num_machines, resources)
    A = []  # Matrice de contraintes
    b = []  # Vecteur des bornes

    # Contraintes CPU
    for j in 1:num_machines
        row = zeros(Int64, num_tasks * num_machines)
        for i in 1:num_tasks
            row[(j-1)*num_tasks + i] = resources[:cpu_req][i]
        end
        push!(A, row)
        push!(b, resources[:cpu_dispo][j])
    end

    # Contraintes RAM
    for j in 1:num_machines
        row = zeros(Int64, num_tasks * num_machines)
        for i in 1:num_tasks
            row[(j-1)*num_tasks + i] = resources[:ram_req][i]
        end
        push!(A, row)
        push!(b, resources[:ram_dispo][j])
    end

    # Contraintes Threads, Bandwidth, IO, Température - Idem

    return A, b
end

# Fonction principale pour résoudre le problème
function solve(num_tasks, num_machines, resources, exec_time)
    A, b = setup_constraints(num_tasks, num_machines, resources)
    println(size(hcat(A)))
    c = vec(exec_time)  # Vectorisation des temps d'exécution

    # Variables initiales avec des bornes (0, 1) pour chaque tâche sur chaque machine
    variables = [(0, 1) for _ in 1:num_tasks * num_machines]

    # Résoudre avec Branch and Cut
    solution = branch_and_cut(A, b, c, variables)

    return solution
end

# Exemple d'utilisation
num_tasks = 3
num_machines = 2
resources = Dict(
    :cpu_req => [2, 3, 1],
    :ram_req => [1, 2, 1],
    :threads_req => [2, 3, 2],
    :bw_req => [10, 15, 5],
    :io_req => [100, 200, 50],
    :temp_req => [10, 15, 5],
    :cpu_dispo => [6, 8],
    :ram_dispo => [8, 10],
    :threads_dispo => [5, 6],
    :bw_dispo => [50, 70],
    :io_dispo => [400, 500],
    :temp_max => [90, 100],
    :temp_current => [30, 40]
)
exec_time = [5 3; 2 6; 4 2]

# Résoudre
solution = solve(num_tasks, num_machines, resources, exec_time)
if solution !== nothing
    println("Meilleure solution trouvée: ", solution)
else
    println("Aucune solution trouvée.")
end